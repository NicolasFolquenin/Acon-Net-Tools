package models

import "time"

// MetricPayloadInput representa o contrato exato recebido via POST /api/v1/metrics
type MetricPayloadInput struct {
	NodeID       string    `json:"node_id"`
	Location     string    `json:"location"`
	Timestamp    time.Time `json:"timestamp"`
	DownloadMbps float64   `json:"download_mbps"`
	UploadMbps   float64   `json:"upload_mbps"`
	PingGateway  float64   `json:"ping_gateway"`
}

// NodeMetadata representa os metadados do nó indexados no Time Series MongoDB
type NodeMetadata struct {
	NodeID   string `bson:"node_id" json:"node_id"`
	Location string `bson:"location" json:"location"`
}

// TimeSeriesMetricDocument é a estrutura persistida na Time Series collection do MongoDB
type TimeSeriesMetricDocument struct {
	Timestamp    time.Time    `bson:"timestamp" json:"timestamp"`
	Metadata     NodeMetadata `bson:"metadata" json:"metadata"`
	DownloadMbps float64      `bson:"download_mbps" json:"download_mbps"`
	UploadMbps   float64      `bson:"upload_mbps" json:"upload_mbps"`
	PingGateway  float64      `bson:"ping_gateway" json:"ping_gateway"`
}

// FlattenedMetricItem representa o item retornado para o dashboard web
type FlattenedMetricItem struct {
	NodeID       string    `json:"node_id"`
	Location     string    `json:"location"`
	Timestamp    time.Time `json:"timestamp"`
	DownloadMbps float64   `json:"download_mbps"`
	UploadMbps   float64   `json:"upload_mbps"`
	PingGateway  float64   `json:"ping_gateway"`
}

// NodeSummaryCard resume a última leitura de cada nó/data center para os cards do dashboard
type NodeSummaryCard struct {
	NodeID          string    `json:"node_id"`
	Location        string    `json:"location"`
	LastSeen        time.Time `json:"last_seen"`
	LastDownload    float64   `json:"last_download"`
	LastUpload      float64   `json:"last_upload"`
	LastPing        float64   `json:"last_ping"`
	IsHealthy       bool      `json:"is_healthy"`
	TotalReadings24 int       `json:"total_readings_24h"`
}

// DashboardResponse resposta agregada do endpoint GET /api/v1/dashboard/metrics
type DashboardResponse struct {
	TotalNodes    int                   `json:"total_nodes"`
	NodesList     []string              `json:"nodes_list"`
	Summaries     []NodeSummaryCard     `json:"summaries"`
	MetricsSeries []FlattenedMetricItem `json:"metrics_series"`
	GeneratedAt   time.Time             `json:"generated_at"`
}
