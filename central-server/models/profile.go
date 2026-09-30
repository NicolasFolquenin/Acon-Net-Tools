package models

import "time"

// TaskType define os tipos de tarefas suportadas pela sonda
type TaskType string

const (
	TaskTypePing               TaskType = "ping"
	TaskTypePacketLoss         TaskType = "packet_loss"
	TaskTypeThroughputDownload TaskType = "http_throughput_download"
	TaskTypeThroughputUpload   TaskType = "http_throughput_upload"
	TaskTypeDNSLookup          TaskType = "dns_lookup"
	TaskTypeDiscoveryProtocol  TaskType = "discovery_protocol"
	TaskTypeSystemInfo         TaskType = "system_info"
)

// TestTaskConfig define a configuração de um teste individual enviado à sonda
type TestTaskConfig struct {
	ID                   string                 `bson:"id" json:"id"`
	Type                 TaskType               `bson:"type" json:"type"`
	Name                 string                 `bson:"name" json:"name"`
	Enabled              bool                   `bson:"enabled" json:"enabled"`
	Target               string                 `bson:"target,omitempty" json:"target,omitempty"`                             // Ex: "auto_gateway", "177.101.22.40"
	Endpoint             string                 `bson:"endpoint,omitempty" json:"endpoint,omitempty"`                         // Ex: "/api/v1/speedtest/download"
	SizeMB               int                    `bson:"size_mb,omitempty" json:"size_mb,omitempty"`                           // Ex: 5
	Domain               string                 `bson:"domain,omitempty" json:"domain,omitempty"`                             // Ex: "uepg.br"
	Count                int                    `bson:"count,omitempty" json:"count,omitempty"`                               // Ex: 3
	TimeoutMS            int                    `bson:"timeout_ms,omitempty" json:"timeout_ms,omitempty"`                     // Ex: 1000
	TimeoutSeconds       int                    `bson:"timeout_seconds,omitempty" json:"timeout_seconds,omitempty"`           // Ex: 15
	Protocol             string                 `bson:"protocol,omitempty" json:"protocol,omitempty"`                         // Ex: "LLDP_CDP"
	ListenTimeoutSeconds int                    `bson:"listen_timeout_seconds,omitempty" json:"listen_timeout_seconds,omitempty"` // Ex: 45
	ExtraParams          map[string]interface{} `bson:"extra_params,omitempty" json:"extra_params,omitempty"`
}

// MeasurementProfile representa o manifesto de perfil de medição distribuído para as sondas
type MeasurementProfile struct {
	Version         string           `bson:"version" json:"version"`
	UpdatedAt       time.Time        `bson:"updated_at" json:"updated_at"`
	IntervalMinutes int              `bson:"interval_minutes" json:"interval_minutes"`
	SyncSchedule    string           `bson:"sync_schedule" json:"sync_schedule"` // Ex: "fixed_half_hour" ou "interval"
	Description     string           `bson:"description,omitempty" json:"description,omitempty"`
	Tests           []TestTaskConfig `bson:"tests" json:"tests"`
}

// DynamicMetricResult representa o resultado padronizado de um teste executado
type DynamicMetricResult struct {
	Value   float64     `json:"value"`
	Unit    string      `json:"unit"`
	Status  string      `json:"status"` // "ok", "warning", "error", "skipped"
	Details interface{} `json:"details,omitempty"`
	Error   string      `json:"error,omitempty"`
}
