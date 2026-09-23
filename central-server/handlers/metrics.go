package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
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
	hoursLimit := h.cfg.QueryHoursLimit

	ctx, cancel := contextWithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Busca séries temporais das últimas 24 horas
	series, err := h.db.GetMetricsLast24Hours(ctx, nodeFilter, hoursLimit)
	if err != nil {
		log.Printf("[API][ERRO] Falha na consulta de dashboard: %v", err)
		// Se banco estiver vazio ou off, retorna payload estruturado vazio
		series = []models.FlattenedMetricItem{}
	}

	// Lista de nós distintos
	nodesList, _ := h.db.GetDistinctNodes(ctx)
	if len(nodesList) == 0 {
		// Detecta nós a partir da série em memória se o distinct falhar
		nodeMap := make(map[string]bool)
		for _, s := range series {
			nodeMap[s.NodeID] = true
		}
		for n := range nodeMap {
			nodesList = append(nodesList, n)
		}
	}
	sort.Strings(nodesList)

	// Agrega resumos de cada nó (última leitura, taxa de sucesso, etc.)
	summaries := computeNodeSummaries(series)

	resp := models.DashboardResponse{
		TotalNodes:    len(nodesList),
		NodesList:     nodesList,
		Summaries:     summaries,
		MetricsSeries: series,
		GeneratedAt:   time.Now().UTC(),
	}

	_ = json.NewEncoder(w).Encode(resp)
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
			if cur.knownSw == nil || item.Timestamp.After(cur.latest.Timestamp) {
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
		if sw == nil || !sw.IsValid() {
			if agg.knownSw != nil {
				sw = agg.knownSw
			}
		}

		summaries = append(summaries, models.NodeSummaryCard{
			NodeID:          agg.latest.NodeID,
			Location:        agg.latest.Location,
			LastSeen:        agg.latest.Timestamp,
			LastDownload:    agg.latest.DownloadMbps,
			LastUpload:      agg.latest.UploadMbps,
			LastPing:        agg.latest.PingGateway,
			IsHealthy:       isHealthy,
			TotalReadings24: agg.count,
			SystemInfo:      sys,
			SwitchInfo:      sw,
		})
	}

	// Ordena por NodeID alfabeticamente
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].NodeID < summaries[j].NodeID
	})

	return summaries
}

func contextWithTimeout(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, d)
}
