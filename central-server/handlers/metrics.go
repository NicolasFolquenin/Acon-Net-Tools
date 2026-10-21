package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"central-server/config"
	"central-server/db"
	"central-server/models"
)

// MetricsHandler gerencia os endpoints relacionados à telemetria de rede
type MetricsHandler struct {
	db  *db.MongoDBClient
	cfg *config.AppConfig
}

// NewMetricsHandler cria uma nova instância de MetricsHandler
func NewMetricsHandler(database *db.MongoDBClient, cfg *config.AppConfig) *MetricsHandler {
	return &MetricsHandler{
		db:  database,
		cfg: cfg,
	}
}

// PostMetric lida com POST /api/v1/metrics
func (h *MetricsHandler) PostMetric(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "metodo nao permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var payload models.MetricPayloadInput
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&payload); err != nil {
		log.Printf("[API][ERRO] Payload invalido: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":   "corpo da requisicao invalido",
			"details": err.Error(),
		})
		return
	}

	// Validações dos campos obrigatórios
	if payload.NodeID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "campo 'node_id' e obrigatorio"})
		return
	}
	if payload.Location == "" {
		payload.Location = "Desconhecido"
	}
	if payload.Timestamp.IsZero() {
		payload.Timestamp = time.Now().UTC()
	}

	// Insere na coleção Time Series do MongoDB
	ctx, cancel := contextWithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.db.InsertMetric(ctx, &payload); err != nil {
		log.Printf("[API][ERRO] Falha ao persistir no MongoDB: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "falha ao salvar metrica no banco de dados",
		})
		return
	}

	log.Printf("[API][SUCESSO] Metrica gravada com sucesso: No=%s, Download=%.1f Mbps, Ping=%.1f ms",
		payload.NodeID, payload.DownloadMbps, payload.PingGateway)

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "success",
		"message":   "Metrica armazenada com sucesso",
		"timestamp": payload.Timestamp,
	})
}

// GetDashboardMetrics lida com GET /api/v1/dashboard/metrics
// Retorna as métricas de todos os nós das últimas 24 horas ordenadas por timestamp
func (h *MetricsHandler) GetDashboardMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "metodo nao permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	nodeFilter := r.URL.Query().Get("node_id")
	dateFilter := r.URL.Query().Get("date")
	hoursLimit := 720 // Padrão: 30 dias (720h) para garantir que histórico recente e medições anteriores nunca sejam excluídos
	if hoursParam := r.URL.Query().Get("hours"); hoursParam != "" {
		if hoursParam == "all" || hoursParam == "0" {
			hoursLimit = 0 // Sem limite temporal
		} else if hVal, err := strconv.Atoi(hoursParam); err == nil && hVal > 0 {
			hoursLimit = hVal
		}
	} else if h.cfg.QueryHoursLimit > 0 && h.cfg.QueryHoursLimit != 24 {
		hoursLimit = h.cfg.QueryHoursLimit
	}

	ctx, cancel := contextWithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Busca séries temporais com suporte a hours e date no MongoDB
	series, err := h.db.GetMetricsByQuery(ctx, nodeFilter, hoursLimit, dateFilter)
	if err != nil {
		log.Printf("[API][ERRO] Falha na consulta de dashboard: %v", err)
		series = []models.FlattenedMetricItem{}
	}

	var isHistoricalFallback bool
	var diagnostic *models.NodeDiagnosticInfo

	// Se estiver filtrando por um nó específico
	if nodeFilter != "" && nodeFilter != "all" {
		// Se não houver medições nas últimas 24h, busca as últimas medições históricas gravadas
		if len(series) == 0 {
			historical, errHist := h.db.GetHistoricalMetricsForNode(ctx, nodeFilter, 20)
			if errHist == nil && len(historical) > 0 {
				series = historical
				isHistoricalFallback = true
			}
		}

		// Constrói diagnóstico detalhado da sonda
		diagnostic = h.buildNodeDiagnostic(ctx, nodeFilter, series, isHistoricalFallback)
	}

	// Agrega resumos de cada nó (última leitura, taxa de sucesso, etc.) com desduplicação por MAC
	summaries := computeNodeSummaries(series)

	// Lista de nós distintos baseada nos nós desduplicados
	nodesList := make([]string, 0, len(summaries))
	for _, s := range summaries {
		nodesList = append(nodesList, s.NodeID)
	}
	sort.Strings(nodesList)

	// Ajusta status de saúde do resumo com base no diagnóstico preciso
	if diagnostic != nil {
		foundSummary := false
		for i := range summaries {
			if summaries[i].NodeID == diagnostic.NodeID {
				summaries[i].IsHealthy = diagnostic.IsOnline
				foundSummary = true
				break
			}
		}
		if !foundSummary && len(series) > 0 {
			latestItem := series[len(series)-1]
			vlan := latestItem.VLAN
			if vlan == 0 && diagnostic.SwitchInfo != nil && diagnostic.SwitchInfo.VLAN > 0 {
				vlan = diagnostic.SwitchInfo.VLAN
			}
			summaries = append(summaries, models.NodeSummaryCard{
				NodeID:          diagnostic.NodeID,
				Location:        diagnostic.Location,
				GroupName:       diagnostic.GroupName,
				VLAN:            vlan,
				LastSeen:        latestItem.Timestamp,
				LastDownload:    latestItem.DownloadMbps,
				LastUpload:      latestItem.UploadMbps,
				LastPing:        latestItem.PingGateway,
				ProfileVersion:  latestItem.ProfileVersion,
				Metrics:         latestItem.Metrics,
				IsHealthy:       diagnostic.IsOnline,
				TotalReadings24: len(series),
				SystemInfo:      diagnostic.SystemInfo,
				SwitchInfo:      diagnostic.SwitchInfo,
				ModularResults:  latestItem.ModularResults,
			})
		}
	}

	resp := models.DashboardResponse{
		TotalNodes:     len(nodesList),
		NodesList:      nodesList,
		Summaries:      summaries,
		MetricsSeries:  series,
		GeneratedAt:    time.Now().UTC(),
		NodeDiagnostic: diagnostic,
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// buildNodeDiagnostic calcula detalhadamente o estado de conexão e perda de sinal de uma máquina
func (h *MetricsHandler) buildNodeDiagnostic(ctx context.Context, nodeID string, series []models.FlattenedMetricItem, isHistorical bool) *models.NodeDiagnosticInfo {
	if nodeID == "" || nodeID == "all" {
		return nil
	}

	diag := &models.NodeDiagnosticInfo{
		NodeID:               nodeID,
		IsHistoricalFallback: isHistorical,
		ExpectedIntervalMin:  30,
	}

	// Consulta metadados da máquina cadastrada
	mach, _ := h.db.GetMachine(ctx, nodeID)
	if mach != nil {
		if mach.GroupID != "" {
			grp, _ := h.db.GetMachineGroup(ctx, mach.GroupID)
			if grp != nil {
				diag.GroupName = grp.Name
				if grp.Profile.IntervalMinutes > 0 {
					diag.ExpectedIntervalMin = grp.Profile.IntervalMinutes
				}
			}
		}
	}

	// Localiza última leitura gravada
	var latest *models.FlattenedMetricItem
	if len(series) > 0 {
		latest = &series[len(series)-1]
	} else {
		latestFromDB, errLatest := h.db.GetLatestMetricForNode(ctx, nodeID)
		if errLatest == nil && latestFromDB != nil {
			latest = latestFromDB
		}
	}

	if latest != nil {
		diag.Location = latest.Location
		if diag.GroupName == "" {
			diag.GroupName = latest.GroupName
		}
		diag.VLAN = latest.VLAN
		if diag.VLAN == 0 && latest.SwitchInfo != nil && latest.SwitchInfo.VLAN > 0 {
			diag.VLAN = latest.SwitchInfo.VLAN
		}
		diag.SystemInfo = latest.SystemInfo
		if diag.SystemInfo == nil && mach != nil && mach.IPAddress != "" {
			diag.SystemInfo = &models.SystemInfo{
				Hostname:   mach.Hostname,
				IPAddress:  mach.IPAddress,
				MACAddress: mach.MACAddress,
			}
		}
		diag.SwitchInfo = latest.SwitchInfo
		if (diag.SwitchInfo == nil || !diag.SwitchInfo.IsValid()) && mach != nil && mach.SwitchName != "" {
			diag.SwitchInfo = &models.SwitchInfo{
				SwitchName: mach.SwitchName,
				SwitchPort: mach.SwitchPort,
				VLAN:       mach.VLAN,
			}
		}

		lastTs := latest.Timestamp.UTC()
		diag.LastMetricTimestamp = &lastTs

		// O ciclo subsequente esperado era lastTs + ExpectedIntervalMin
		lostAt := lastTs.Add(time.Duration(diag.ExpectedIntervalMin) * time.Minute)
		now := time.Now().UTC()

		// Tolerância de 5 minutos antes de classificar como offline
		graceLimit := lostAt.Add(5 * time.Minute)
		if now.After(graceLimit) {
			diag.IsOnline = false
			diag.LostConnectionAt = &lostAt
			downtime := now.Sub(lostAt)
			if downtime < 0 {
				downtime = 0
			}
			diag.DowntimeSeconds = int64(downtime.Seconds())
			diag.DowntimeDuration = formatDowntime(downtime)
		} else {
			diag.IsOnline = true
			diag.LostConnectionAt = nil
			diag.DowntimeSeconds = 0
			diag.DowntimeDuration = ""
		}
	} else {
		// Nenhuma leitura jamais recebida
		diag.IsOnline = false
		diag.DowntimeDuration = "Sem telemetria registrada"
		if mach != nil {
			diag.VLAN = mach.VLAN
			if mach.IPAddress != "" {
				diag.SystemInfo = &models.SystemInfo{
					Hostname:   mach.Hostname,
					IPAddress:  mach.IPAddress,
					MACAddress: mach.MACAddress,
				}
			}
			if mach.SwitchName != "" {
				diag.SwitchInfo = &models.SwitchInfo{
					SwitchName: mach.SwitchName,
					SwitchPort: mach.SwitchPort,
					VLAN:       mach.VLAN,
				}
			}
		}
	}

	return diag
}

// formatDowntime formata a duração do tempo offline de forma amigável em português
func formatDowntime(d time.Duration) string {
	if d <= 0 {
		return "poucos instantes"
	}
	totalMinutes := int64(d.Minutes())
	if totalMinutes < 60 {
		return fmt.Sprintf("%d min", totalMinutes)
	}
	totalHours := totalMinutes / 60
	remMinutes := totalMinutes % 60
	if totalHours < 24 {
		if remMinutes == 0 {
			return fmt.Sprintf("%d h", totalHours)
		}
		return fmt.Sprintf("%d h e %d min", totalHours, remMinutes)
	}
	days := totalHours / 24
	remHours := totalHours % 24
	if remHours == 0 {
		if days == 1 {
			return "1 dia"
		}
		return fmt.Sprintf("%d dias", days)
	}
	if days == 1 {
		return fmt.Sprintf("1 dia e %d h", remHours)
	}
	return fmt.Sprintf("%d dias e %d h", days, remHours)
}


// DeleteNode lida com DELETE /api/v1/nodes?node_id={id} para remover um computador do sistema
func (h *MetricsHandler) DeleteNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		http.Error(w, `{"error": "metodo nao permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	nodeID := strings.TrimSpace(r.URL.Query().Get("node_id"))
	if nodeID == "" {
		// Tenta ler do body caso não esteja na query string
		var bodyReq struct {
			NodeID string `json:"node_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&bodyReq); err == nil {
			nodeID = strings.TrimSpace(bodyReq.NodeID)
		}
	}

	if nodeID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "parametro 'node_id' e obrigatorio",
		})
		return
	}

	ctx, cancel := contextWithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	count, err := h.db.DeleteNode(ctx, nodeID)
	if err != nil {
		log.Printf("[API][ERRO] Falha ao excluir no '%s': %v", nodeID, err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "falha ao excluir no do banco de dados",
		})
		return
	}

	log.Printf("[API] No '%s' excluido com sucesso pelo administrador (%d registros removidos)", nodeID, count)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "success",
		"message":       fmt.Sprintf("Computador '%s' e suas metricas foram excluidos com sucesso", nodeID),
		"node_id":       nodeID,
		"deleted_count": count,
	})
}

// computeNodeSummaries agrupa por nó e localiza a leitura mais recente
func computeNodeSummaries(series []models.FlattenedMetricItem) []models.NodeSummaryCard {
	type nodeAgg struct {
		latest   models.FlattenedMetricItem
		count    int
		knownSys *models.SystemInfo
		knownSw  *models.SwitchInfo
	}

	aggMap := make(map[string]*nodeAgg)

	for _, item := range series {
		cur, exists := aggMap[item.NodeID]
		if !exists {
			cur = &nodeAgg{latest: item, count: 1}
			aggMap[item.NodeID] = cur
		} else {
			cur.count++
			if item.Timestamp.After(cur.latest.Timestamp) {
				cur.latest = item
			}
		}

		if item.SystemInfo != nil && (cur.knownSys == nil || item.Timestamp.After(cur.latest.Timestamp)) {
			cur.knownSys = item.SystemInfo
		}
		if item.SwitchInfo != nil && item.SwitchInfo.IsValid() {
			if item.SwitchInfo.IsRealSwitch() {
				if cur.knownSw == nil || !cur.knownSw.IsRealSwitch() || item.Timestamp.After(cur.latest.Timestamp) {
					cur.knownSw = item.SwitchInfo
				}
			} else if cur.knownSw == nil {
				cur.knownSw = item.SwitchInfo
			}
		}
	}

	var summaries []models.NodeSummaryCard
	for _, agg := range aggMap {
		isHealthy := time.Since(agg.latest.Timestamp) < 3*time.Hour
		sys := agg.latest.SystemInfo
		if sys == nil {
			sys = agg.knownSys
		}
		sw := agg.latest.SwitchInfo
		if sw == nil || !sw.IsRealSwitch() {
			if agg.knownSw != nil && agg.knownSw.IsRealSwitch() {
				sw = agg.knownSw
			} else if sw == nil || !sw.IsValid() {
				sw = agg.knownSw
			}
		}

		vlan := agg.latest.VLAN
		if vlan == 0 && sw != nil && sw.VLAN > 0 {
			vlan = sw.VLAN
		}

		summaries = append(summaries, models.NodeSummaryCard{
			NodeID:          agg.latest.NodeID,
			Location:        agg.latest.Location,
			GroupName:       agg.latest.GroupName,
			VLAN:            vlan,
			LastSeen:        agg.latest.Timestamp,
			LastDownload:    agg.latest.DownloadMbps,
			LastUpload:      agg.latest.UploadMbps,
			LastPing:        agg.latest.PingGateway,
			ProfileVersion:  agg.latest.ProfileVersion,
			Metrics:         agg.latest.Metrics,
			IsHealthy:       isHealthy,
			TotalReadings24: agg.count,
			SystemInfo:      sys,
			SwitchInfo:      sw,
			ModularResults:  agg.latest.ModularResults,
		})
	}

	// Desduplicação inteligente por MAC Address:
	// Máquinas renomeadas possuem o mesmo endereço MAC físico.
	// O nó com timestamp de telemetria mais recente prevalece como nó ativo.
	macMap := make(map[string]models.NodeSummaryCard)
	var nonMacSummaries []models.NodeSummaryCard

	for _, s := range summaries {
		mac := ""
		if s.SystemInfo != nil {
			mac = normalizeMAC(s.SystemInfo.MACAddress)
		}
		if isValidMAC(mac) {
			if existing, found := macMap[mac]; !found {
				macMap[mac] = s
			} else {
				if s.LastSeen.After(existing.LastSeen) {
					macMap[mac] = s
				}
			}
		} else {
			nonMacSummaries = append(nonMacSummaries, s)
		}
	}

	deduped := nonMacSummaries
	for _, s := range macMap {
		deduped = append(deduped, s)
	}

	// Ordena por NodeID alfabeticamente
	sort.Slice(deduped, func(i, j int) bool {
		return deduped[i].NodeID < deduped[j].NodeID
	})

	return deduped
}

func normalizeMAC(mac string) string {
	mac = strings.ToUpper(strings.TrimSpace(mac))
	mac = strings.ReplaceAll(mac, "-", ":")
	return mac
}

func isValidMAC(mac string) bool {
	norm := normalizeMAC(mac)
	if norm == "" || norm == "00:00:00:00:00:00" || norm == "FF:FF:FF:FF:FF:FF" {
		return false
	}
	parts := strings.Split(norm, ":")
	if len(parts) != 6 {
		return false
	}
	for _, p := range parts {
		if len(p) != 2 {
			return false
		}
	}
	return true
}

func contextWithTimeout(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, d)
}
