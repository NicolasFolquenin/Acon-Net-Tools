package models

import (
	"strings"
	"time"
)

// SystemInfo armazena dados de hardware e rede local do host
type SystemInfo struct {
	Hostname       string  `bson:"hostname,omitempty" json:"hostname,omitempty"`
	Model          string  `bson:"model,omitempty" json:"model,omitempty"`
	CPU            string  `bson:"cpu,omitempty" json:"cpu,omitempty"`
	RAMTotalGB     float64 `bson:"ram_total_gb,omitempty" json:"ram_total_gb,omitempty"`
	OS             string  `bson:"os,omitempty" json:"os,omitempty"`
	IPAddress      string  `bson:"ip_address,omitempty" json:"ip_address,omitempty"`
	MACAddress     string  `bson:"mac_address,omitempty" json:"mac_address,omitempty"`
	Gateway        string  `bson:"gateway,omitempty" json:"gateway,omitempty"`
	NetworkAdapter string  `bson:"network_adapter,omitempty" json:"network_adapter,omitempty"`
}

// SwitchInfo armazena as informações do switch descobertas via LLDP
type SwitchInfo struct {
	SwitchName   string    `bson:"switch_name,omitempty" json:"switch_name,omitempty"`
	SwitchPort   string    `bson:"switch_port,omitempty" json:"switch_port,omitempty"`
	VLAN         int       `bson:"vlan,omitempty" json:"vlan,omitempty"`
	SwitchModel  string    `bson:"switch_model,omitempty" json:"switch_model,omitempty"`
	Protocol     string    `bson:"protocol,omitempty" json:"protocol,omitempty"`
	DiscoveredAt time.Time `bson:"discovered_at,omitempty" json:"discovered_at,omitempty"`
}

// IsValid verifica se o switch possui identificação real e válida descoberta por LLDP/CDP
func (s *SwitchInfo) IsValid() bool {
	if s == nil {
		return false
	}
	name := strings.TrimSpace(s.SwitchName)
	if name == "" || name == "--" || name == "{}" {
		return false
	}
	lower := strings.ToLower(name)
	if strings.HasPrefix(lower, "aguardando") {
		return false
	}
	if strings.HasPrefix(lower, "não detectado") || strings.HasPrefix(lower, "nao detectado") {
		return false
	}
	return true
}

// IsRealSwitch indica se a informação é de fato um switch de rede e não um telefone IP/VoIP
func (s *SwitchInfo) IsRealSwitch() bool {
	if !s.IsValid() {
		return false
	}
	name := strings.ToUpper(strings.TrimSpace(s.SwitchName))
	if strings.HasPrefix(name, "SEP") && len(name) == 15 {
		return false
	}
	model := strings.ToLower(s.SwitchModel)
	if strings.Contains(model, "phone") || strings.Contains(model, "telefone") || strings.Contains(model, "voip") {
		return false
	}
	return true
}

// MetricPayloadInput representa o contrato exato recebido via POST /api/v1/metrics
type MetricPayloadInput struct {
	NodeID       string      `json:"node_id"`
	Location     string      `json:"location"`
	Timestamp    time.Time   `json:"timestamp"`
	DownloadMbps float64     `json:"download_mbps"`
	UploadMbps   float64     `json:"upload_mbps"`
	PingGateway  float64     `json:"ping_gateway"`
	SystemInfo   *SystemInfo `json:"system_info,omitempty"`
	SwitchInfo   *SwitchInfo `json:"switch_info,omitempty"`
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
	SystemInfo   *SystemInfo  `bson:"system_info,omitempty" json:"system_info,omitempty"`
	SwitchInfo   *SwitchInfo  `bson:"switch_info,omitempty" json:"switch_info,omitempty"`
}

// FlattenedMetricItem representa o item retornado para o dashboard web
type FlattenedMetricItem struct {
	NodeID       string      `json:"node_id"`
	Location     string      `json:"location"`
	Timestamp    time.Time   `json:"timestamp"`
	DownloadMbps float64     `json:"download_mbps"`
	UploadMbps   float64     `json:"upload_mbps"`
	PingGateway  float64     `json:"ping_gateway"`
	SystemInfo   *SystemInfo `json:"system_info,omitempty"`
	SwitchInfo   *SwitchInfo `json:"switch_info,omitempty"`
}

// NodeSummaryCard resume a última leitura de cada nó para os cards do dashboard
type NodeSummaryCard struct {
	NodeID          string      `json:"node_id"`
	Location        string      `json:"location"`
	LastSeen        time.Time   `json:"last_seen"`
	LastDownload    float64     `json:"last_download"`
	LastUpload      float64     `json:"last_upload"`
	LastPing        float64     `json:"last_ping"`
	IsHealthy       bool        `json:"is_healthy"`
	TotalReadings24 int         `json:"total_readings_24h"`
	SystemInfo      *SystemInfo `json:"system_info,omitempty"`
	SwitchInfo      *SwitchInfo `json:"switch_info,omitempty"`
}

// DashboardResponse resposta agregada do endpoint GET /api/v1/dashboard/metrics
type DashboardResponse struct {
	TotalNodes    int                   `json:"total_nodes"`
	NodesList     []string              `json:"nodes_list"`
	Summaries     []NodeSummaryCard     `json:"summaries"`
	MetricsSeries []FlattenedMetricItem `json:"metrics_series"`
	GeneratedAt   time.Time             `json:"generated_at"`
}
