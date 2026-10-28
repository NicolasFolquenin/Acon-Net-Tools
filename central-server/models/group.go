package models

import "time"

// PingModuleConfig configurações para teste de ping
type PingModuleConfig struct {
	Enabled   bool   `json:"enabled" bson:"enabled"`
	Target    string `json:"target" bson:"target"`
	Count     int    `json:"count" bson:"count"`
	TimeoutMs int    `json:"timeout_ms" bson:"timeout_ms"`
}

// PacketLossModuleConfig configurações para teste de perda de pacotes e jitter
type PacketLossModuleConfig struct {
	Enabled   bool   `json:"enabled" bson:"enabled"`
	Target    string `json:"target" bson:"target"`
	Count     int    `json:"count" bson:"count"`
	TimeoutMs int    `json:"timeout_ms" bson:"timeout_ms"`
}

// SpeedtestDownloadConfig configurações para teste de download
type SpeedtestDownloadConfig struct {
	Enabled  bool  `json:"enabled" bson:"enabled"`
	MaxBytes int64 `json:"max_bytes" bson:"max_bytes"`
}

// SpeedtestUploadConfig configurações para teste de upload
type SpeedtestUploadConfig struct {
	Enabled  bool  `json:"enabled" bson:"enabled"`
	MaxBytes int64 `json:"max_bytes" bson:"max_bytes"`
}

// DNSModuleConfig configurações para teste de resolução de nomes
type DNSModuleConfig struct {
	Enabled bool     `json:"enabled" bson:"enabled"`
	Domains []string `json:"domains" bson:"domains"`
}

// HTTPCheckModuleConfig configurações para teste de disponibilidade de endpoint HTTP
type HTTPCheckModuleConfig struct {
	Enabled        bool   `json:"enabled" bson:"enabled"`
	URL            string `json:"url" bson:"url"`
	ExpectedStatus int    `json:"expected_status" bson:"expected_status"`
}

// TracerouteModuleConfig configurações para traceroute / saltos de rede
type TracerouteModuleConfig struct {
	Enabled bool   `json:"enabled" bson:"enabled"`
	Target  string `json:"target" bson:"target"`
	MaxHops int    `json:"max_hops" bson:"max_hops"`
}

// LLDPModuleConfig configurações para descoberta LLDP
type LLDPModuleConfig struct {
	Enabled bool `json:"enabled" bson:"enabled"`
}

// TestModulesConfig agrega todos os módulos de teste configuráveis
type TestModulesConfig struct {
	Ping       PingModuleConfig        `json:"ping" bson:"ping"`
	PacketLoss PacketLossModuleConfig  `json:"packet_loss" bson:"packet_loss"`
	Download   SpeedtestDownloadConfig `json:"download" bson:"download"`
	Upload     SpeedtestUploadConfig   `json:"upload" bson:"upload"`
	DNS        DNSModuleConfig         `json:"dns" bson:"dns"`
	HTTPCheck  HTTPCheckModuleConfig   `json:"http_check" bson:"http_check"`
	Traceroute TracerouteModuleConfig  `json:"traceroute" bson:"traceroute"`
	LLDP       LLDPModuleConfig        `json:"lldp" bson:"lldp"`
}

// DefaultModulesConfig retorna a configuração padrão de módulos com valores sensatos
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
			MaxBytes: 10485760, // 10MB
		},
		Upload: SpeedtestUploadConfig{
			Enabled:  true,
			MaxBytes: 5242880, // 5MB
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

// TestGroup representa um grupo de testes com VLANs vinculadas
type TestGroup struct {
	ID              string            `json:"id" bson:"_id,omitempty"`
	Name            string            `json:"name" bson:"name"`
	Description     string            `json:"description" bson:"description"`
	VLANs           []int             `json:"vlans" bson:"vlans"`
	IsDefault       bool              `json:"is_default" bson:"is_default"`
	IntervalMinutes int               `json:"interval_minutes" bson:"interval_minutes"`
	Modules         TestModulesConfig `json:"modules" bson:"modules"`
	CreatedAt       time.Time         `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at" bson:"updated_at"`
}

// ProbeConfigResponse resposta enviada quando a sonda consulta suas diretrizes
type ProbeConfigResponse struct {
	GroupID         string            `json:"group_id"`
	GroupName       string            `json:"group_name"`
	MatchedVLAN     int               `json:"matched_vlan"`
	IsDefault       bool              `json:"is_default"`
	IntervalMinutes int               `json:"interval_minutes"`
	Modules         TestModulesConfig `json:"modules"`
}
