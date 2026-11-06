package models

import (
	"strings"
	"time"
)

// NetworkInterfaceDetail representa uma placa de rede identificada no computador
type NetworkInterfaceDetail struct {
	Name        string            `bson:"name" json:"name"`                                   // Ex: "Wi-Fi", "Ethernet", "eth0"
	Description string            `bson:"description,omitempty" json:"description,omitempty"` // Ex: "Intel(R) Wi-Fi 6 AX201"
	MACAddress  string            `bson:"mac_address" json:"mac_address"`                     // Ex: "E4-54-E8-12-34-56"
	IPAddresses []string          `bson:"ip_addresses,omitempty" json:"ip_addresses,omitempty"`
	Type        string            `bson:"type" json:"type"`                                   // "ethernet", "wifi", "virtual", "other"
	IsPrimary   bool              `bson:"is_primary" json:"is_primary"`                       // Placa com a rota default/gateway ativo
	IsUp        bool              `bson:"is_up" json:"is_up"`                                 // Link operacional ativo
	Gateway     string            `bson:"gateway,omitempty" json:"gateway,omitempty"`
	IsWireless  bool              `bson:"is_wireless,omitempty" json:"is_wireless,omitempty"`
	WiFiInfo    *WiFiMetricResult `bson:"wifi_info,omitempty" json:"wifi_info,omitempty"`
}

// UserAccountInfo representa os dados de uma conta de usuário do sistema operacional
type UserAccountInfo struct {
	Username    string `bson:"username" json:"username"`
	FullName    string `bson:"full_name,omitempty" json:"full_name,omitempty"`
	Enabled     bool   `bson:"enabled" json:"enabled"`
	IsAdmin     bool   `bson:"is_admin,omitempty" json:"is_admin,omitempty"`
	IsLogged    bool   `bson:"is_logged,omitempty" json:"is_logged,omitempty"`
	Description string `bson:"description,omitempty" json:"description,omitempty"`
}

// SystemInfo armazena dados de hardware e rede local do host
type SystemInfo struct {
	Hostname       string                   `bson:"hostname,omitempty" json:"hostname,omitempty"`
	Model          string                   `bson:"model,omitempty" json:"model,omitempty"`
	CPU            string                   `bson:"cpu,omitempty" json:"cpu,omitempty"`
	RAMTotalGB     float64                  `bson:"ram_total_gb,omitempty" json:"ram_total_gb,omitempty"`
	OS             string                   `bson:"os,omitempty" json:"os,omitempty"`
	IPAddress      string                   `bson:"ip_address,omitempty" json:"ip_address,omitempty"`
	MACAddress     string                   `bson:"mac_address,omitempty" json:"mac_address,omitempty"`
	Gateway        string                   `bson:"gateway,omitempty" json:"gateway,omitempty"`
	NetworkAdapter string                   `bson:"network_adapter,omitempty" json:"network_adapter,omitempty"`
	Interfaces     []NetworkInterfaceDetail `bson:"interfaces,omitempty" json:"interfaces,omitempty"`
	ClientVersion  string                   `bson:"client_version,omitempty" json:"client_version,omitempty"`
	LoggedUser     string                   `bson:"logged_user,omitempty" json:"logged_user,omitempty"` // Usuário da sessão/processo que envia as medições
	ActiveUser     string                   `bson:"active_user,omitempty" json:"active_user,omitempty"` // Usuário interativo do console/desktop
	Users          []UserAccountInfo        `bson:"users,omitempty" json:"users,omitempty"`             // Contas de usuário locais da máquina
	WiFiInfo       *WiFiMetricResult        `bson:"wifi_info,omitempty" json:"wifi_info,omitempty"`
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
	// Telefones VoIP da Cisco não são switches de rede
	upper := strings.ToUpper(name)
	if strings.HasPrefix(upper, "SEP") && len(upper) == 15 {
		return false
	}
	model := strings.ToLower(s.SwitchModel)
	if strings.Contains(model, "phone") || strings.Contains(model, "telefone") || strings.Contains(model, "voip") {
		return false
	}
	return true
}

// IsRealSwitch é um alias semântico de IsValid, mantido para legibilidade no domínio de rede.
// No contexto de diagnóstico, "real switch" comunica a intenção mais claramente do que "valid"
// ao distinguir switches corporativos (Cisco, HP, etc.) de telefones VoIP ou dados ausentes.
func (s *SwitchInfo) IsRealSwitch() bool {
	return s.IsValid()
}

// DNSDomainResult armazena a resolução de um domínio específico
type DNSDomainResult struct {
	Domain     string  `bson:"domain" json:"domain"`
	ResolvedIP string  `bson:"resolved_ip" json:"resolved_ip"`
	DurationMs float64 `bson:"duration_ms" json:"duration_ms"`
	Success    bool    `bson:"success" json:"success"`
	Error      string  `bson:"error,omitempty" json:"error,omitempty"`
}

// DNSResult armazena o resultado do teste de DNS
type DNSResult struct {
	Domains []DNSDomainResult `bson:"domains" json:"domains"`
	AvgMs   float64           `bson:"avg_ms" json:"avg_ms"`
}

// PacketLossResult armazena perda de pacotes e jitter
type PacketLossResult struct {
	Target       string  `bson:"target" json:"target"`
	Sent         int     `bson:"sent" json:"sent"`
	Received     int     `bson:"received" json:"received"`
	LossPercent  float64 `bson:"loss_percent" json:"loss_percent"`
	JitterMs     float64 `bson:"jitter_ms" json:"jitter_ms"`
	MinLatencyMs float64 `bson:"min_latency_ms" json:"min_latency_ms"`
	AvgLatencyMs float64 `bson:"avg_latency_ms" json:"avg_latency_ms"`
	MaxLatencyMs float64 `bson:"max_latency_ms" json:"max_latency_ms"`
}

// HTTPCheckResult armazena resultado do teste de healthcheck HTTP
type HTTPCheckResult struct {
	URL        string  `bson:"url" json:"url"`
	StatusCode int     `bson:"status_code" json:"status_code"`
	DurationMs float64 `bson:"duration_ms" json:"duration_ms"`
	Success    bool    `bson:"success" json:"success"`
	Error      string  `bson:"error,omitempty" json:"error,omitempty"`
}

// TracerouteHop armazena um salto de rota
type TracerouteHop struct {
	HopNum    int     `bson:"hop_num" json:"hop_num"`
	IP        string  `bson:"ip" json:"ip"`
	LatencyMs float64 `bson:"latency_ms" json:"latency_ms"`
}

// TracerouteResult armazena o resultado do teste de saltos
type TracerouteResult struct {
	Target    string          `bson:"target" json:"target"`
	TotalHops int             `bson:"total_hops" json:"total_hops"`
	Hops      []TracerouteHop `bson:"hops" json:"hops"`
	Success   bool            `bson:"success" json:"success"`
}

type SingleDNSProbeResult struct {
	Server     string  `bson:"server" json:"server"`
	LatencyMs  float64 `bson:"latency_ms" json:"latency_ms"`
	ResolvedIP string  `bson:"resolved_ip" json:"resolved_ip"`
	Success    bool    `bson:"success" json:"success"`
	Error      string  `bson:"error,omitempty" json:"error,omitempty"`
}

type MultiDNSResult struct {
	Domain  string                 `bson:"domain" json:"domain"`
	Servers []SingleDNSProbeResult `bson:"servers" json:"servers"`
	BestMs  float64                `bson:"best_ms" json:"best_ms"`
	AvgMs   float64                `bson:"avg_ms" json:"avg_ms"`
}

type WiFiMetricResult struct {
	HasWiFiAdapter bool    `bson:"has_wifi_adapter" json:"has_wifi_adapter"`
	Connected      bool    `bson:"connected" json:"connected"`
	SSID           string  `bson:"ssid,omitempty" json:"ssid,omitempty"`
	BSSID          string  `bson:"bssid,omitempty" json:"bssid,omitempty"`
	SignalPercent  int     `bson:"signal_percent" json:"signal_percent"`
	SignalDBm      int     `bson:"signal_dbm" json:"signal_dbm"`
	RadioType      string  `bson:"radio_type,omitempty" json:"radio_type,omitempty"`
	Band           string  `bson:"band,omitempty" json:"band,omitempty"`
	Channel        int     `bson:"channel,omitempty" json:"channel,omitempty"`
	RxRateMbps     float64 `bson:"rx_rate_mbps,omitempty" json:"rx_rate_mbps,omitempty"`
	TxRateMbps     float64 `bson:"tx_rate_mbps,omitempty" json:"tx_rate_mbps,omitempty"`
	Status         string  `bson:"status" json:"status"`
}

// ModularResults agrupa os resultados dos novos módulos de testes
type ModularResults struct {
	DNS        *DNSResult        `bson:"dns,omitempty" json:"dns,omitempty"`
	MultiDNS   *MultiDNSResult   `bson:"multi_dns,omitempty" json:"multi_dns,omitempty"`
	WiFi       *WiFiMetricResult `bson:"wifi,omitempty" json:"wifi,omitempty"`
	PacketLoss *PacketLossResult `bson:"packet_loss,omitempty" json:"packet_loss,omitempty"`
	HTTPCheck  *HTTPCheckResult  `bson:"http_check,omitempty" json:"http_check,omitempty"`
	Traceroute *TracerouteResult `bson:"traceroute,omitempty" json:"traceroute,omitempty"`
}

// MetricPayloadInput representa o contrato exato recebido via POST /api/v1/metrics
type MetricPayloadInput struct {
	NodeID         string                 `json:"node_id"`
	Location       string                 `json:"location"`
	GroupName      string                 `json:"group_name,omitempty"`
	VLAN           int                    `json:"vlan,omitempty"`
	Timestamp      time.Time              `json:"timestamp"`
	DownloadMbps   float64                `json:"download_mbps"`
	UploadMbps     float64                `json:"upload_mbps"`
	PingGateway    float64                `json:"ping_gateway"`
	ProfileVersion string                 `json:"profile_version,omitempty"`
	ClientVersion  string                 `json:"client_version,omitempty"`
	LoggedUser     string                 `json:"logged_user,omitempty"`
	Metrics        map[string]interface{} `json:"metrics,omitempty"`
	SystemInfo     *SystemInfo            `json:"system_info,omitempty"`
	SwitchInfo     *SwitchInfo            `json:"switch_info,omitempty"`
	ModularResults *ModularResults        `json:"modular_results,omitempty"`
}

// NodeMetadata representa os metadados do nó indexados no Time Series MongoDB
type NodeMetadata struct {
	NodeID    string `bson:"node_id" json:"node_id"`
	Location  string `bson:"location" json:"location"`
	GroupName string `bson:"group_name,omitempty" json:"group_name,omitempty"`
	VLAN      int    `bson:"vlan,omitempty" json:"vlan,omitempty"`
}

// TimeSeriesMetricDocument é a estrutura persistida na Time Series collection do MongoDB
type TimeSeriesMetricDocument struct {
	Timestamp      time.Time              `bson:"timestamp" json:"timestamp"`
	Metadata       NodeMetadata           `bson:"metadata" json:"metadata"`
	DownloadMbps   float64                `bson:"download_mbps" json:"download_mbps"`
	UploadMbps     float64                `bson:"upload_mbps" json:"upload_mbps"`
	PingGateway    float64                `bson:"ping_gateway" json:"ping_gateway"`
	ProfileVersion string                 `bson:"profile_version,omitempty" json:"profile_version,omitempty"`
	ClientVersion  string                 `bson:"client_version,omitempty" json:"client_version,omitempty"`
	LoggedUser     string                 `bson:"logged_user,omitempty" json:"logged_user,omitempty"`
	Metrics        map[string]interface{} `bson:"metrics,omitempty" json:"metrics,omitempty"`
	SystemInfo     *SystemInfo            `bson:"system_info,omitempty" json:"system_info,omitempty"`
	SwitchInfo     *SwitchInfo            `bson:"switch_info,omitempty" json:"switch_info,omitempty"`
	ModularResults *ModularResults        `bson:"modular_results,omitempty" json:"modular_results,omitempty"`
}

// FlattenedMetricItem representa o item retornado para o dashboard web
type FlattenedMetricItem struct {
	NodeID         string                 `json:"node_id"`
	Location       string                 `json:"location"`
	GroupName      string                 `json:"group_name,omitempty"`
	VLAN           int                    `json:"vlan,omitempty"`
	Timestamp      time.Time              `json:"timestamp"`
	DownloadMbps   float64                `json:"download_mbps"`
	UploadMbps     float64                `json:"upload_mbps"`
	PingGateway    float64                `json:"ping_gateway"`
	ProfileVersion string                 `json:"profile_version,omitempty"`
	ClientVersion  string                 `json:"client_version,omitempty"`
	LoggedUser     string                 `json:"logged_user,omitempty"`
	Metrics        map[string]interface{} `json:"metrics,omitempty"`
	SystemInfo     *SystemInfo            `json:"system_info,omitempty"`
	SwitchInfo     *SwitchInfo            `json:"switch_info,omitempty"`
	ModularResults *ModularResults        `json:"modular_results,omitempty"`
}

// NodeSummaryCard resume a última leitura de cada nó para os cards do dashboard
type NodeSummaryCard struct {
	NodeID          string                 `json:"node_id"`
	Location        string                 `json:"location"`
	GroupName       string                 `json:"group_name,omitempty"`
	VLAN            int                    `json:"vlan,omitempty"`
	LastSeen        time.Time              `json:"last_seen"`
	LastDownload    float64                `json:"last_download"`
	LastUpload      float64                `json:"last_upload"`
	LastPing        float64                `json:"last_ping"`
	ProfileVersion  string                 `json:"profile_version,omitempty"`
	ClientVersion   string                 `json:"client_version,omitempty"`
	LoggedUser      string                 `json:"logged_user,omitempty"`
	Metrics         map[string]interface{} `json:"metrics,omitempty"`
	IsHealthy       bool                   `json:"is_healthy"`
	TotalReadings24 int                    `json:"total_readings_24h"`
	SystemInfo      *SystemInfo            `json:"system_info,omitempty"`
	SwitchInfo      *SwitchInfo            `json:"switch_info,omitempty"`
	ModularResults  *ModularResults        `json:"modular_results,omitempty"`
}

// NodeDiagnosticInfo traz detalhes sobre o estado de conexão e perda de sinal de uma máquina específica
type NodeDiagnosticInfo struct {
	NodeID               string      `json:"node_id"`
	IsOnline             bool        `json:"is_online"`
	LastMetricTimestamp  *time.Time  `json:"last_metric_timestamp,omitempty"`
	LostConnectionAt     *time.Time  `json:"lost_connection_at,omitempty"`
	DowntimeDuration     string      `json:"downtime_duration,omitempty"`
	DowntimeSeconds      int64       `json:"downtime_seconds,omitempty"`
	ExpectedIntervalMin  int         `json:"expected_interval_minutes,omitempty"`
	IsHistoricalFallback bool        `json:"is_historical_fallback"`
	Location             string      `json:"location,omitempty"`
	GroupName            string      `json:"group_name,omitempty"`
	VLAN                 int         `json:"vlan,omitempty"`
	ClientVersion        string      `json:"client_version,omitempty"`
	SystemInfo           *SystemInfo `json:"system_info,omitempty"`
	SwitchInfo           *SwitchInfo     `json:"switch_info,omitempty"`
	ModularResults       *ModularResults `json:"modular_results,omitempty"`
}

// DashboardResponse resposta agregada do endpoint GET /api/v1/dashboard/metrics
type DashboardResponse struct {
	TotalNodes     int                   `json:"total_nodes"`
	NodesList      []string              `json:"nodes_list"`
	Summaries      []NodeSummaryCard     `json:"summaries"`
	MetricsSeries  []FlattenedMetricItem `json:"metrics_series"`
	GeneratedAt    time.Time             `json:"generated_at"`
	NodeDiagnostic *NodeDiagnosticInfo   `json:"node_diagnostic,omitempty"`
}
