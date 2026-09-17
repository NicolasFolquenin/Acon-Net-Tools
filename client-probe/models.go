package main

import "time"

// MetricPayload representa o contrato oficial enviado via POST HTTPS ao servidor central.
type MetricPayload struct {
	NodeID       string    `json:"node_id"`
	Location     string    `json:"location"`
	Timestamp    time.Time `json:"timestamp"`
	DownloadMbps float64   `json:"download_mbps"`
	UploadMbps   float64   `json:"upload_mbps"`
	PingGateway  float64   `json:"ping_gateway"`
}

// LocalMetricRecord armazena a medição no SQLite local, incluindo status de sincronização.
type LocalMetricRecord struct {
	ID           int64     `json:"id"`
	NodeID       string    `json:"node_id"`
	Location     string    `json:"location"`
	Timestamp    time.Time `json:"timestamp"`
	DownloadMbps float64   `json:"download_mbps"`
	UploadMbps   float64   `json:"upload_mbps"`
	PingGateway  float64   `json:"ping_gateway"`
	Synced       bool      `json:"synced"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// Config representa as configurações carregadas de config.json
type Config struct {
	NodeID          string `json:"node_id"`
	Location        string `json:"location"`
	ServerURL       string `json:"server_url"`
	APIToken        string `json:"api_token"`
	IntervalMinutes int    `json:"interval_minutes"`
	PingTarget      string `json:"ping_target"`
}

// ProbeStatus informa o estado atual da sonda para a interface visual.
type ProbeStatus struct {
	IsRunning     bool               `json:"is_running"`
	StatusText    string             `json:"status_text"`
	LastRun       *time.Time         `json:"last_run,omitempty"`
	NextRun       *time.Time         `json:"next_run,omitempty"`
	LastMetric    *LocalMetricRecord `json:"last_metric,omitempty"`
	ServerStatus  string             `json:"server_status"`
	TotalSynced   int                `json:"total_synced"`
	TotalPending  int                `json:"total_pending"`
}
