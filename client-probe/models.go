package main

import "time"

// SystemInfo armazena dados de hardware e rede local da máquina que executa o teste
type SystemInfo struct {
	Hostname       string  `json:"hostname"`
	Model          string  `json:"model"`
	CPU            string  `json:"cpu"`
	RAMTotalGB     float64 `json:"ram_total_gb"`
	OS             string  `json:"os"`
	IPAddress      string  `json:"ip_address"`
	MACAddress     string  `json:"mac_address"`
	Gateway        string  `json:"gateway"`
	NetworkAdapter string  `json:"network_adapter"`
}

// SwitchInfo armazena as informações do switch descobertas via LLDP
type SwitchInfo struct {
	SwitchName   string    `json:"switch_name"`
	SwitchPort   string    `json:"switch_port"`
	VLAN         int       `json:"vlan,omitempty"`
	SwitchModel  string    `json:"switch_model,omitempty"`
	Protocol     string    `json:"protocol,omitempty"`
	DiscoveredAt time.Time `json:"discovered_at,omitempty"`
}

// MetricPayload representa o contrato oficial enviado via POST HTTPS ao servidor central.
type MetricPayload struct {
	NodeID       string      `json:"node_id"`
	Location     string      `json:"location"`
	Timestamp    time.Time   `json:"timestamp"`
	DownloadMbps float64     `json:"download_mbps"`
	UploadMbps   float64     `json:"upload_mbps"`
	PingGateway  float64     `json:"ping_gateway"`
	SystemInfo   *SystemInfo `json:"system_info,omitempty"`
	SwitchInfo   *SwitchInfo `json:"switch_info,omitempty"`
}

// LocalMetricRecord armazena a medição no SQLite local, incluindo status de sincronização.
type LocalMetricRecord struct {
	ID           int64       `json:"id"`
	NodeID       string      `json:"node_id"`
	Location     string      `json:"location"`
	Timestamp    time.Time   `json:"timestamp"`
	DownloadMbps float64     `json:"download_mbps"`
	UploadMbps   float64     `json:"upload_mbps"`
	PingGateway  float64     `json:"ping_gateway"`
	Synced       bool        `json:"synced"`
	ErrorMessage string      `json:"error_message,omitempty"`
	SystemInfo   *SystemInfo `json:"system_info,omitempty"`
	SwitchInfo   *SwitchInfo `json:"switch_info,omitempty"`
}

// Config representa as configurações carregadas de config.json / .env
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
	CurrentPC     *SystemInfo        `json:"current_pc,omitempty"`
	CurrentSwitch *SwitchInfo        `json:"current_switch,omitempty"`
}
