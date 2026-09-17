package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sort"
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

// SeedSampleData popula o MongoDB com dados realistas de exemplo caso o banco esteja vazio
func (h *MetricsHandler) SeedSampleData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	nodes := []struct {
		ID       string
		Location string
		BaseDown float64
		BaseUp   float64
		BasePing float64
	}{
		{"minipc-dc-01", "Guarapuava-DC1", 850.0, 720.0, 1.2},
		{"minipc-dc-02", "Curitiba-DC2", 920.0, 810.0, 3.4},
		{"minipc-dc-03", "SaoPaulo-DC3", 780.0, 690.0, 8.5},
	}

	now := time.Now().UTC()
	ctx := r.Context()
	totalInserted := 0

	for _, n := range nodes {
		for i := 23; i >= 0; i-- {
			t := now.Add(-time.Duration(i) * time.Hour)
			down := n.BaseDown + float64((i*7)%35) - 15.0
			up := n.BaseUp + float64((i*5)%25) - 10.0
			ping := n.BasePing + float64((i*3)%10)*0.1

			_ = h.db.InsertMetric(ctx, &models.MetricPayloadInput{
				NodeID:       n.ID,
				Location:     n.Location,
				Timestamp:    t,
				DownloadMbps: down,
				UploadMbps:   up,
				PingGateway:  ping,
			})
			totalInserted++
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "seeded",
		"total_inserted": totalInserted,
		"message":        "Dados simulados de 24 horas gerados com sucesso!",
	})
}

// computeNodeSummaries agrupa por nó e localiza a leitura mais recente
func computeNodeSummaries(series []models.FlattenedMetricItem) []models.NodeSummaryCard {
	type nodeAgg struct {
		latest models.FlattenedMetricItem
		count  int
	}

	aggMap := make(map[string]*nodeAgg)

	for _, item := range series {
		if cur, exists := aggMap[item.NodeID]; !exists {
			aggMap[item.NodeID] = &nodeAgg{latest: item, count: 1}
		} else {
			cur.count++
			if item.Timestamp.After(cur.latest.Timestamp) {
				cur.latest = item
			}
		}
	}

	var summaries []models.NodeSummaryCard
	for _, agg := range aggMap {
		isHealthy := time.Since(agg.latest.Timestamp) < 3*time.Hour
		summaries = append(summaries, models.NodeSummaryCard{
			NodeID:          agg.latest.NodeID,
			Location:        agg.latest.Location,
			LastSeen:        agg.latest.Timestamp,
			LastDownload:    agg.latest.DownloadMbps,
			LastUpload:      agg.latest.UploadMbps,
			LastPing:        agg.latest.PingGateway,
			IsHealthy:       isHealthy,
			TotalReadings24: agg.count,
			SystemInfo:      agg.latest.SystemInfo,
			SwitchInfo:      agg.latest.SwitchInfo,
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
