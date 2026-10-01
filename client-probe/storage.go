package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type StorageService struct {
	db         *sql.DB
	httpClient *http.Client
	mu         sync.Mutex
}

const dbFilename = "probe_local.db"

// NewStorageService inicializa a conexão com o banco SQLite local probe_local.db.
func NewStorageService() (*StorageService, error) {
	db, err := sql.Open("sqlite", dbFilename)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir sqlite: %w", err)
	}

	// Otimizações de concorrência para SQLite
	db.SetMaxOpenConns(1)

	service := &StorageService{
		db: db,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}

	if err := service.initSchema(); err != nil {
		return nil, err
	}

	return service, nil
}

func (s *StorageService) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		node_id TEXT NOT NULL,
		location TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		download_mbps REAL NOT NULL,
		upload_mbps REAL NOT NULL,
		ping_gateway REAL NOT NULL,
		synced INTEGER NOT NULL DEFAULT 0,
		error_message TEXT,
		system_info_json TEXT,
		switch_info_json TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics (timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_metrics_synced ON metrics (synced);
	`
	_, err := s.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("falha ao criar schema sqlite: %w", err)
	}

	// Migrações dinâmicas para bases existentes
	_, _ = s.db.Exec(`ALTER TABLE metrics ADD COLUMN system_info_json TEXT;`)
	_, _ = s.db.Exec(`ALTER TABLE metrics ADD COLUMN switch_info_json TEXT;`)
	_, _ = s.db.Exec(`ALTER TABLE metrics ADD COLUMN profile_version TEXT;`)
	_, _ = s.db.Exec(`ALTER TABLE metrics ADD COLUMN metrics_json TEXT;`)
	_, _ = s.db.Exec(`ALTER TABLE metrics ADD COLUMN group_name TEXT;`)
	_, _ = s.db.Exec(`ALTER TABLE metrics ADD COLUMN vlan INTEGER;`)
	_, _ = s.db.Exec(`ALTER TABLE metrics ADD COLUMN modular_json TEXT;`)

	log.Printf("[STORAGE] Banco SQLite (%s) inicializado com sucesso", dbFilename)
	return nil
}

// SaveMetric insere uma nova leitura no banco local.
func (s *StorageService) SaveMetric(m *MetricPayload, synced bool, errMsg string) (*LocalMetricRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	syncedInt := 0
	if synced {
		syncedInt = 1
	}

	var sysJson, switchJson, metricsJson, modularJson string
	if m.SystemInfo != nil {
		if b, err := json.Marshal(m.SystemInfo); err == nil {
			sysJson = string(b)
		}
	}
	if m.SwitchInfo != nil {
		if b, err := json.Marshal(m.SwitchInfo); err == nil {
			switchJson = string(b)
		}
	}
	if m.Metrics != nil {
		if b, err := json.Marshal(m.Metrics); err == nil {
			metricsJson = string(b)
		}
	}
	if m.ModularResults != nil {
		if b, err := json.Marshal(m.ModularResults); err == nil {
			modularJson = string(b)
		}
	}

	query := `
	INSERT INTO metrics (node_id, location, group_name, vlan, timestamp, download_mbps, upload_mbps, ping_gateway, synced, error_message, system_info_json, switch_info_json, profile_version, metrics_json, modular_json)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	res, err := s.db.Exec(query,
		m.NodeID,
		m.Location,
		m.GroupName,
		m.VLAN,
		m.Timestamp.Format(time.RFC3339),
		m.DownloadMbps,
		m.UploadMbps,
		m.PingGateway,
		syncedInt,
		errMsg,
		sysJson,
		switchJson,
		m.ProfileVersion,
		metricsJson,
		modularJson,
	)
	if err != nil {
		return nil, fmt.Errorf("erro ao salvar metrica no sqlite: %w", err)
	}

	id, _ := res.LastInsertId()
	return &LocalMetricRecord{
		ID:             id,
		NodeID:         m.NodeID,
		Location:       m.Location,
		GroupName:      m.GroupName,
		VLAN:           m.VLAN,
		Timestamp:      m.Timestamp,
		DownloadMbps:   m.DownloadMbps,
		UploadMbps:     m.UploadMbps,
		PingGateway:    m.PingGateway,
		ProfileVersion: m.ProfileVersion,
		Metrics:        m.Metrics,
		Synced:         synced,
		ErrorMessage:   errMsg,
		SystemInfo:     m.SystemInfo,
		SwitchInfo:     m.SwitchInfo,
		ModularResults: m.ModularResults,
	}, nil
}

// GetRecentMetrics retorna as últimas N execuções
func (s *StorageService) GetRecentMetrics(limit int) ([]LocalMetricRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit <= 0 {
		limit = 24
	}

	query := `
	SELECT id, node_id, location, COALESCE(group_name, ''), COALESCE(vlan, 0), timestamp, download_mbps, upload_mbps, ping_gateway, synced, 
	       COALESCE(error_message, ''), COALESCE(system_info_json, ''), COALESCE(switch_info_json, ''),
	       COALESCE(profile_version, ''), COALESCE(metrics_json, ''), COALESCE(modular_json, '')
	FROM metrics
	ORDER BY timestamp DESC
	LIMIT ?
	`
	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar metricas: %w", err)
	}
	defer rows.Close()

	var descList []LocalMetricRecord
	for rows.Next() {
		var r LocalMetricRecord
		var timeStr string
		var syncedInt int
		var sysJson, switchJson, profileVer, metricsJson, modularJson string
		if err := rows.Scan(&r.ID, &r.NodeID, &r.Location, &r.GroupName, &r.VLAN, &timeStr, &r.DownloadMbps, &r.UploadMbps, &r.PingGateway, &syncedInt, &r.ErrorMessage, &sysJson, &switchJson, &profileVer, &metricsJson, &modularJson); err != nil {
			log.Printf("[STORAGE] Erro scan: %v", err)
			continue
		}
		t, _ := time.Parse(time.RFC3339, timeStr)
		r.Timestamp = t
		r.Synced = (syncedInt == 1)
		r.ProfileVersion = profileVer

		if sysJson != "" {
			var si SystemInfo
			if errU := json.Unmarshal([]byte(sysJson), &si); errU == nil {
				r.SystemInfo = &si
			}
		}
		if switchJson != "" {
			var sw SwitchInfo
			if errU := json.Unmarshal([]byte(switchJson), &sw); errU == nil {
				r.SwitchInfo = &sw
			}
		}
		if metricsJson != "" {
			var m map[string]interface{}
			if errU := json.Unmarshal([]byte(metricsJson), &m); errU == nil {
				r.Metrics = m
			}
		}
		if modularJson != "" {
			var mr ModularResults
			if errU := json.Unmarshal([]byte(modularJson), &mr); errU == nil {
				r.ModularResults = &mr
			}
		}

		descList = append(descList, r)
	}

	// Inverte para ficar cronológico (mais antigo -> mais recente)
	ascList := make([]LocalMetricRecord, len(descList))
	for i, item := range descList {
		ascList[len(descList)-1-i] = item
	}

	return ascList, nil
}

// GetLastValidSwitchInfo procura nas últimas medições pelo switch válido mais recente gravado, priorizando switches reais
func (s *StorageService) GetLastValidSwitchInfo() *SwitchInfo {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
	SELECT switch_info_json 
	FROM metrics 
	WHERE switch_info_json IS NOT NULL AND switch_info_json != '' AND switch_info_json != '{}'
	ORDER BY timestamp DESC 
	LIMIT 30
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var fallback *SwitchInfo
	for rows.Next() {
		var switchJson string
		if err := rows.Scan(&switchJson); err == nil && switchJson != "" {
			var sw SwitchInfo
			if errU := json.Unmarshal([]byte(switchJson), &sw); errU == nil {
				if sw.IsRealSwitch() {
					return &sw
				}
				if sw.IsValid() && fallback == nil {
					fallback = &sw
				}
			}
		}
	}
	return fallback
}

// GetCounts retorna o total de registros sincronizados e pendentes.
func (s *StorageService) GetCounts() (synced int, pending int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_ = s.db.QueryRow("SELECT COUNT(*) FROM metrics WHERE synced = 1").Scan(&synced)
	_ = s.db.QueryRow("SELECT COUNT(*) FROM metrics WHERE synced = 0").Scan(&pending)
	return synced, pending
}

// SyncPendingMetrics busca leituras pendentes (synced = 0) e tenta reenviá-las ao servidor
func (s *StorageService) SyncPendingMetrics(ctx context.Context, serverURL, apiToken string) int {
	s.mu.Lock()
	rows, err := s.db.Query(`
		SELECT id, node_id, location, COALESCE(group_name, ''), COALESCE(vlan, 0), timestamp, download_mbps, upload_mbps, ping_gateway, 
		       COALESCE(system_info_json, ''), COALESCE(switch_info_json, ''),
		       COALESCE(profile_version, ''), COALESCE(metrics_json, ''), COALESCE(modular_json, '')
		FROM metrics WHERE synced = 0 ORDER BY id ASC LIMIT 20
	`)
	if err != nil {
		s.mu.Unlock()
		return 0
	}
	defer rows.Close()

	type pendingItem struct {
		id int64
		p  MetricPayload
	}
	var list []pendingItem
	for rows.Next() {
		var item pendingItem
		var timeStr, sysJson, swJson, profileVer, metricsJson, modularJson string
		if err := rows.Scan(&item.id, &item.p.NodeID, &item.p.Location, &item.p.GroupName, &item.p.VLAN, &timeStr, &item.p.DownloadMbps, &item.p.UploadMbps, &item.p.PingGateway, &sysJson, &swJson, &profileVer, &metricsJson, &modularJson); err != nil {
			continue
		}
		item.p.Timestamp, _ = time.Parse(time.RFC3339, timeStr)
		item.p.ProfileVersion = profileVer

		if sysJson != "" {
			var si SystemInfo
			if errU := json.Unmarshal([]byte(sysJson), &si); errU == nil {
				item.p.SystemInfo = &si
			}
		}
		if swJson != "" {
			var sw SwitchInfo
			if errU := json.Unmarshal([]byte(swJson), &sw); errU == nil {
				item.p.SwitchInfo = &sw
			}
		}
		if metricsJson != "" {
			var m map[string]interface{}
			if errU := json.Unmarshal([]byte(metricsJson), &m); errU == nil {
				item.p.Metrics = m
			}
		}
		if modularJson != "" {
			var mr ModularResults
			if errU := json.Unmarshal([]byte(modularJson), &mr); errU == nil {
				item.p.ModularResults = &mr
			}
		}
		list = append(list, item)
	}
	s.mu.Unlock()

	syncedCount := 0
	for _, item := range list {
		if err := s.SendToServer(ctx, &item.p, serverURL, apiToken); err == nil {
			s.mu.Lock()
			_, _ = s.db.Exec("UPDATE metrics SET synced = 1, error_message = '' WHERE id = ?", item.id)
			s.mu.Unlock()
			syncedCount++
		}
	}
	if syncedCount > 0 {
		log.Printf("[STORAGE][SYNC] %d metricas pendentes foram sincronizadas com sucesso!", syncedCount)
	}
	return syncedCount
}

// SendToServer envia a medição via POST HTTPS para o servidor central.
func (s *StorageService) SendToServer(ctx context.Context, payload *MetricPayload, serverURL, apiToken string) error {
	endpoint := strings.TrimRight(serverURL, "/") + "/api/v1/metrics"

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("falha na serializacao json: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("falha ao criar requisicao http: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiToken)

	log.Printf("[STORAGE] Enviando metricas enriquecidas para %s...", endpoint)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("[STORAGE][AVISO] Falha de conexao com servidor central (%s): %v", endpoint, err)
		return fmt.Errorf("servidor inalcancavel: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("status code %d: %s", resp.StatusCode, string(bodyBytes))

		// Se o servidor for uma versão legada que usa DisallowUnknownFields, reenvia sem os metadados extras
		if resp.StatusCode == 400 && strings.Contains(string(bodyBytes), "unknown field") && (payload.SystemInfo != nil || payload.SwitchInfo != nil) {
			log.Printf("[STORAGE][COMPATIBILIDADE] Servidor central em versao legada. Reenviando payload basico para garantir sincronizacao...")
			fallbackPayload := *payload
			fallbackPayload.SystemInfo = nil
			fallbackPayload.SwitchInfo = nil
			return s.SendToServer(ctx, &fallbackPayload, serverURL, apiToken)
		}

		log.Printf("[STORAGE][AVISO] Servidor central recusou envio: %s", errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	log.Printf("[STORAGE][SUCESSO] Metricas sincronizadas com servidor central (HTTP %d)", resp.StatusCode)
	return nil
}

// Close fecha a conexão SQLite
func (s *StorageService) Close() {
	if s.db != nil {
		_ = s.db.Close()
	}
}
