package main

import (
	"strings"
	"time"
)

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

// TaskType define os tipos de tarefas de medição suportadas
type TaskType string

const (
	TaskTypePing               TaskType = "ping"
	TaskTypePacketLoss         TaskType = "packet_loss"
	TaskTypeThroughputDownload TaskType = "http_throughput_download"
	TaskTypeThroughputUpload   TaskType = "http_throughput_upload"
	TaskTypeDNSLookup          TaskType = "dns_lookup"
	TaskTypeDiscoveryProtocol  TaskType = "discovery_protocol"
	TaskTypeSystemInfo         TaskType = "system_info"
	TaskTypeHTTPCheck          TaskType = "http_check"
	TaskTypeTraceroute         TaskType = "traceroute"
)

// HostBootstrapInfo carrega dados do host para o auto-registro inicial
type HostBootstrapInfo struct {
	Hostname   string `json:"hostname"`
	IPAddress  string `json:"ip_address"`
	MACAddress string `json:"mac_address"`
	VLAN       int    `json:"vlan"`
	SwitchName string `json:"switch_name"`
	SwitchIP   string `json:"switch_ip"`
	SwitchPort string `json:"switch_port"`
}

// ProbeRegisterRequest representa o payload enviado no POST /api/v1/probes/register
type ProbeRegisterRequest struct {
	Hostname   string `json:"hostname"`
	IPAddress  string `json:"ip_address"`
	MACAddress string `json:"mac_address"`
	VLAN       int    `json:"vlan"`
	SwitchName string `json:"switch_name"`
	SwitchIP   string `json:"switch_ip"`
	SwitchPort string `json:"switch_port"`
}

// TestTaskConfig define os parâmetros de uma medição específica recebida do servidor central
type TestTaskConfig struct {
	ID                   string                 `json:"id"`
	Type                 TaskType               `json:"type"`
	Name                 string                 `json:"name"`
	Enabled              bool                   `json:"enabled"`
	Target               string                 `json:"target,omitempty"`
	Endpoint             string                 `json:"endpoint,omitempty"`
	SizeMB               int                    `json:"size_mb,omitempty"`
	Domain               string                 `json:"domain,omitempty"`
	Count                int                    `json:"count,omitempty"`
	TimeoutMS            int                    `json:"timeout_ms,omitempty"`
	TimeoutSeconds       int                    `json:"timeout_seconds,omitempty"`
	Protocol             string                 `json:"protocol,omitempty"`
	ListenTimeoutSeconds int                    `json:"listen_timeout_seconds,omitempty"`
	ExtraParams          map[string]interface{} `json:"extra_params,omitempty"`
}

// MeasurementProfile representa o manifesto de perfil de medição distribuído pelo servidor central
type MeasurementProfile struct {
	Version         string           `json:"version"`
	UpdatedAt       time.Time        `json:"updated_at"`
	IntervalMinutes int              `json:"interval_minutes"`
	SyncSchedule    string           `json:"sync_schedule"` // Ex: "fixed_half_hour" ou "interval"
	Description     string           `json:"description,omitempty"`
	Tests           []TestTaskConfig `json:"tests"`
}

// DynamicMetricResult armazena o resultado individual de uma métrica executada
type DynamicMetricResult struct {
	Value   float64     `json:"value"`
	Unit    string      `json:"unit"`
	Status  string      `json:"status"` // "ok", "warning", "error", "skipped"
	Details interface{} `json:"details,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Configurações dos Módulos Individuais
type PingModuleConfig struct {
	Enabled   bool   `json:"enabled"`
	Target    string `json:"target"`
	Count     int    `json:"count"`
	TimeoutMs int    `json:"timeout_ms"`
}

type PacketLossModuleConfig struct {
	Enabled   bool   `json:"enabled"`
	Target    string `json:"target"`
	Count     int    `json:"count"`
	TimeoutMs int    `json:"timeout_ms"`
}

type SpeedtestDownloadConfig struct {
	Enabled  bool  `json:"enabled"`
	MaxBytes int64 `json:"max_bytes"`
}

type SpeedtestUploadConfig struct {
	Enabled  bool  `json:"enabled"`
	MaxBytes int64 `json:"max_bytes"`
}

type DNSModuleConfig struct {
	Enabled bool     `json:"enabled"`
	Domains []string `json:"domains"`
}

type HTTPCheckModuleConfig struct {
	Enabled        bool   `json:"enabled"`
	URL            string `json:"url"`
	ExpectedStatus int    `json:"expected_status"`
}

type TracerouteModuleConfig struct {
	Enabled bool   `json:"enabled"`
	Target  string `json:"target"`
	MaxHops int    `json:"max_hops"`
}

type LLDPModuleConfig struct {
	Enabled bool `json:"enabled"`
}

type TestModulesConfig struct {
	Ping       PingModuleConfig        `json:"ping"`
	PacketLoss PacketLossModuleConfig  `json:"packet_loss"`
	Download   SpeedtestDownloadConfig `json:"download"`
	Upload     SpeedtestUploadConfig   `json:"upload"`
	DNS        DNSModuleConfig         `json:"dns"`
	HTTPCheck  HTTPCheckModuleConfig   `json:"http_check"`
	Traceroute TracerouteModuleConfig  `json:"traceroute"`
	LLDP       LLDPModuleConfig        `json:"lldp"`
}

func DefaultModulesConfig() TestModulesConfig {
	return TestModulesConfig{
		Ping: PingModuleConfig{
			Enabled:   true,
			Target:    "1.1.1.1",
			Count:     4,
			TimeoutMs: 1000,
		},
		PacketLoss: PacketLossModuleConfig{
			Enabled:   true,
			Target:    "1.1.1.1",
			Count:     10,
			TimeoutMs: 1000,
		},
		Download: SpeedtestDownloadConfig{
			Enabled:  true,
			MaxBytes: 10485760,
		},
		Upload: SpeedtestUploadConfig{
			Enabled:  true,
			MaxBytes: 5242880,
		},
		DNS: DNSModuleConfig{
			Enabled: true,
			Domains: []string{"google.com", "cloudflare.com"},
		},
		HTTPCheck: HTTPCheckModuleConfig{
			Enabled:        true,
			URL:            "https://www.google.com",
			ExpectedStatus: 200,
		},
		Traceroute: TracerouteModuleConfig{
			Enabled: false,
			Target:  "1.1.1.1",
			MaxHops: 15,
		},
		LLDP: LLDPModuleConfig{
			Enabled: true,
		},
	}
}

type ProbeConfigResponse struct {
	GroupID         string            `json:"group_id"`
	GroupName       string            `json:"group_name"`
	MatchedVLAN     int               `json:"matched_vlan"`
	IsDefault       bool              `json:"is_default"`
	IntervalMinutes int               `json:"interval_minutes"`
	Modules         TestModulesConfig `json:"modules"`
}

// Resultados dos Módulos
type DNSDomainResult struct {
	Domain     string  `json:"domain"`
	ResolvedIP string  `json:"resolved_ip"`
	DurationMs float64 `json:"duration_ms"`
	Success    bool    `json:"success"`
	Error      string  `json:"error,omitempty"`
}

type DNSResult struct {
	Domains []DNSDomainResult `json:"domains"`
	AvgMs   float64           `json:"avg_ms"`
}

type PacketLossResult struct {
	Target       string  `json:"target"`
	Sent         int     `json:"sent"`
	Received     int     `json:"received"`
	LossPercent  float64 `json:"loss_percent"`
	JitterMs     float64 `json:"jitter_ms"`
	MinLatencyMs float64 `json:"min_latency_ms"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	MaxLatencyMs float64 `json:"max_latency_ms"`
}

type HTTPCheckResult struct {
	URL        string  `json:"url"`
	StatusCode int     `json:"status_code"`
	DurationMs float64 `json:"duration_ms"`
	Success    bool    `json:"success"`
	Error      string  `json:"error,omitempty"`
}

type TracerouteHop struct {
	HopNum    int     `json:"hop_num"`
	IP        string  `json:"ip"`
	LatencyMs float64 `json:"latency_ms"`
}

type TracerouteResult struct {
	Target    string          `json:"target"`
	TotalHops int             `json:"total_hops"`
	Hops      []TracerouteHop `json:"hops"`
	Success   bool            `json:"success"`
}

type ModularResults struct {
	DNS        *DNSResult        `json:"dns,omitempty"`
	PacketLoss *PacketLossResult `json:"packet_loss,omitempty"`
	HTTPCheck  *HTTPCheckResult  `json:"http_check,omitempty"`
	Traceroute *TracerouteResult `json:"traceroute,omitempty"`
}

// MetricPayload representa o contrato oficial enviado via POST HTTPS ao servidor central.
type MetricPayload struct {
	NodeID         string                 `json:"node_id"`
	Location       string                 `json:"location"`
	GroupName      string                 `json:"group_name,omitempty"`
	VLAN           int                    `json:"vlan,omitempty"`
	Timestamp      time.Time              `json:"timestamp"`
	DownloadMbps   float64                `json:"download_mbps"`
	UploadMbps     float64                `json:"upload_mbps"`
	PingGateway    float64                `json:"ping_gateway"`
	ProfileVersion string                 `json:"profile_version,omitempty"`
	Metrics        map[string]interface{} `json:"metrics,omitempty"`
	SystemInfo     *SystemInfo            `json:"system_info,omitempty"`
	SwitchInfo     *SwitchInfo            `json:"switch_info,omitempty"`
	ModularResults *ModularResults        `json:"modular_results,omitempty"`
}

// LocalMetricRecord armazena a medição no SQLite local, incluindo status de sincronização.
type LocalMetricRecord struct {
	ID             int64                  `json:"id"`
	NodeID         string                 `json:"node_id"`
	Location       string                 `json:"location"`
	GroupName      string                 `json:"group_name,omitempty"`
	VLAN           int                    `json:"vlan,omitempty"`
	Timestamp      time.Time              `json:"timestamp"`
	DownloadMbps   float64                `json:"download_mbps"`
	UploadMbps     float64                `json:"upload_mbps"`
	PingGateway    float64                `json:"ping_gateway"`
	ProfileVersion string                 `json:"profile_version,omitempty"`
	Metrics        map[string]interface{} `json:"metrics,omitempty"`
	Synced         bool                   `json:"synced"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
	SystemInfo     *SystemInfo            `json:"system_info,omitempty"`
	SwitchInfo     *SwitchInfo            `json:"switch_info,omitempty"`
	ModularResults *ModularResults        `json:"modular_results,omitempty"`
}

// Config representa as configurações carregadas de config.json / .env
type Config struct {
	NodeID          string `json:"node_id"`
	Location        string `json:"location"`
	ServerURL       string `json:"server_url"`
	APIToken        string `json:"api_token"`
	IntervalMinutes int    `json:"interval_minutes"`
	PingTarget      string `json:"ping_target"`
	SyncWithCentral bool   `json:"sync_with_central"`
}

// IsCentralSynced retorna se a sonda está configurada para operar sincronizada com o Servidor Central
func (c *Config) IsCentralSynced() bool {
	if c == nil {
		return false
	}
	cleanURL := strings.TrimSpace(c.ServerURL)
	if cleanURL == "" || cleanURL == "none" || cleanURL == "disabled" {
		return false
	}
	return c.SyncWithCentral
}

// ProbeStatus informa o estado atual da sonda para a interface visual.
type ProbeStatus struct {
	IsRunning       bool                `json:"is_running"`
	StatusText      string              `json:"status_text"`
	ActiveTask      string              `json:"active_task,omitempty"`
	GroupName       string              `json:"group_name,omitempty"`
	CurrentVLAN     int                 `json:"current_vlan,omitempty"`
	LastRun         *time.Time          `json:"last_run,omitempty"`
	NextRun         *time.Time          `json:"next_run,omitempty"`
	LastMetric      *LocalMetricRecord  `json:"last_metric,omitempty"`
	ServerStatus    string              `json:"server_status"`
	TotalSynced     int                 `json:"total_synced"`
	TotalPending    int                 `json:"total_pending"`
	CurrentPC       *SystemInfo         `json:"current_pc,omitempty"`
	CurrentSwitch   *SwitchInfo         `json:"current_switch,omitempty"`
	CurrentProfile  *MeasurementProfile `json:"current_profile,omitempty"`
	SyncWithCentral bool                `json:"sync_with_central"`
}
