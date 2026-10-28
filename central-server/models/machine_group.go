package models

import "time"

// GroupAutoRules define as regras para auto-associação de novas máquinas ao grupo
type GroupAutoRules struct {
	VLANs     []int    `bson:"vlans,omitempty" json:"vlans,omitempty"`         // Ex: [10, 11]
	IPSubnets []string `bson:"ip_subnets,omitempty" json:"ip_subnets,omitempty"` // Ex: ["10.10.10.0/24", "177.101.22.0/24"]
}

// MachineGroup representa um agrupamento lógico de máquinas que compartilham um perfil de medição
type MachineGroup struct {
	ID          string             `bson:"_id" json:"id"`                     // Identificador slug: "reitoria", "acon", "default"
	Name        string             `bson:"name" json:"name"`                 // Ex: "Grupo Reitoria"
	Description string             `bson:"description" json:"description"`
	Rules       GroupAutoRules     `bson:"rules" json:"rules"`               // Regras para auto-associação
	Profile     MeasurementProfile `bson:"profile" json:"profile"`           // Perfil com os módulos de medição do grupo
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

// GetInitialGroups retorna os grupos padrão do sistema: default, reitoria e acon
func GetInitialGroups() []MachineGroup {
	now := time.Now().UTC()

	return []MachineGroup{
		{
			ID:          "default",
			Name:        "Grupo Padrão (Bootstrap)",
			Description: "Perfil inicial atribuído a máquinas recém-instaladas sem grupo definido. Executa testes de latência, throughput e DNS.",
			Rules:       GroupAutoRules{}, // Captura quem não bater em nenhuma regra específica
			Profile: MeasurementProfile{
				Version:         "2026.group-default-v2",
				UpdatedAt:       now,
				IntervalMinutes: 30,
				SyncSchedule:    "fixed_half_hour",
				Description:     "Perfil padrão completo: Latência Gateway, Throughput Download & Upload e DNS",
				Tests: []TestTaskConfig{
					{
						ID:             "ping_gateway",
						Type:           TaskTypePing,
						Name:           "Latência Gateway Local",
						Enabled:        true,
						Target:         "auto_gateway",
						Count:          3,
						TimeoutMS:      1000,
						TimeoutSeconds: 15,
					},
					{
						ID:             "throughput_download",
						Type:           TaskTypeThroughputDownload,
						Name:           "Throughput Download",
						Enabled:        true,
						Endpoint:       "/api/v1/speedtest/download",
						SizeMB:         5,
						TimeoutSeconds: 30,
					},
					{
						ID:             "throughput_upload",
						Type:           TaskTypeThroughputUpload,
						Name:           "Throughput Upload",
						Enabled:        true,
						Endpoint:       "/api/v1/speedtest/upload",
						SizeMB:         2,
						TimeoutSeconds: 30,
					},
					{
						ID:             "dns_uepg",
						Type:           TaskTypeDNSLookup,
						Name:           "Resolução DNS Institucional",
						Enabled:        true,
						Domain:         "uepg.br",
						DNSServers:     []string{"auto", "177.101.22.40"},
						TimeoutMS:      2000,
						TimeoutSeconds: 10,
					},
					{
						ID:             "wifi_status",
						Type:           TaskTypeWiFiStatus,
						Name:           "Métricas de Rede Wi-Fi & Sinal",
						Enabled:        true,
						TimeoutSeconds: 10,
					},
				},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:          "reitoria",
			Name:        "Grupo Reitoria",
			Description: "Máquinas administrativas e gabinetes da Reitoria UEPG",
			Rules: GroupAutoRules{
				VLANs:     []int{10, 11},
				IPSubnets: []string{"10.10.10.0/24", "10.10.11.0/24"},
			},
			Profile: MeasurementProfile{
				Version:         "2026.group-reitoria",
				UpdatedAt:       now,
				IntervalMinutes: 30,
				SyncSchedule:    "fixed_half_hour",
				Description:     "Perfil de medição da Reitoria: Gateway, Core NTI, Perda de Pacotes e DNS UEPG",
				Tests: []TestTaskConfig{
					{
						ID:             "ping_gateway",
						Type:           TaskTypePing,
						Name:           "Latência Gateway Local",
						Enabled:        true,
						Target:         "auto_gateway",
						Count:          3,
						TimeoutMS:      1000,
						TimeoutSeconds: 15,
					},
					{
						ID:             "ping_nti_core",
						Type:           TaskTypePing,
						Name:           "Latência Core NTI (Anel)",
						Enabled:        true,
						Target:         "177.101.22.40",
						Count:          3,
						TimeoutMS:      1000,
						TimeoutSeconds: 15,
					},
					{
						ID:             "packet_loss_core",
						Type:           TaskTypePacketLoss,
						Name:           "Perda de Pacotes Core NTI",
						Enabled:        true,
						Target:         "177.101.22.40",
						Count:          10,
						TimeoutMS:      1000,
						TimeoutSeconds: 20,
					},
					{
						ID:             "dns_uepg",
						Type:           TaskTypeDNSLookup,
						Name:           "Resolução DNS Institucional",
						Enabled:        true,
						Domain:         "uepg.br",
						DNSServers:     []string{"auto", "177.101.22.40"},
						TimeoutMS:      2000,
						TimeoutSeconds: 10,
					},
					{
						ID:             "wifi_status",
						Type:           TaskTypeWiFiStatus,
						Name:           "Métricas de Rede Wi-Fi & Sinal",
						Enabled:        true,
						TimeoutSeconds: 10,
					},
				},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:          "acon",
			Name:        "Grupo ACON (Anel de Conectividade)",
			Description: "Sondas dedicadas de alta prioridade instaladas nos blocos do Anel",
			Rules: GroupAutoRules{
				VLANs:     []int{50, 52, 1101},
				IPSubnets: []string{"10.50.0.0/16", "10.0.0.0/16"},
			},
			Profile: MeasurementProfile{
				Version:         "2026.group-acon",
				UpdatedAt:       now,
				IntervalMinutes: 30,
				SyncSchedule:    "fixed_half_hour",
				Description:     "Perfil completo ACON: Ping, Perda de Pacotes, Throughput e LLDP/CDP",
				Tests: []TestTaskConfig{
					{
						ID:             "ping_gateway",
						Type:           TaskTypePing,
						Name:           "Latência Gateway Local",
						Enabled:        true,
						Target:         "auto_gateway",
						Count:          3,
						TimeoutMS:      1000,
						TimeoutSeconds: 15,
					},
					{
						ID:             "ping_nti_core",
						Type:           TaskTypePing,
						Name:           "Latência Core NTI",
						Enabled:        true,
						Target:         "177.101.22.40",
						Count:          3,
						TimeoutMS:      1000,
						TimeoutSeconds: 15,
					},
					{
						ID:             "packet_loss_gateway",
						Type:           TaskTypePacketLoss,
						Name:           "Perda de Pacotes Gateway",
						Enabled:        true,
						Target:         "auto_gateway",
						Count:          10,
						TimeoutMS:      1000,
						TimeoutSeconds: 20,
					},
					{
						ID:             "dns_uepg",
						Type:           TaskTypeDNSLookup,
						Name:           "Resolução DNS Institucional",
						Enabled:        true,
						Domain:         "uepg.br",
						DNSServers:     []string{"auto", "177.101.22.40"},
						TimeoutMS:      2000,
						TimeoutSeconds: 10,
					},
					{
						ID:             "wifi_status",
						Type:           TaskTypeWiFiStatus,
						Name:           "Métricas de Rede Wi-Fi & Sinal",
						Enabled:        true,
						TimeoutSeconds: 10,
					},
					{
						ID:             "throughput_download",
						Type:           TaskTypeThroughputDownload,
						Name:           "Throughput Download ACON",
						Enabled:        true,
						Endpoint:       "/api/v1/speedtest/download",
						SizeMB:         5,
						TimeoutSeconds: 45,
					},
					{
						ID:             "throughput_upload",
						Type:           TaskTypeThroughputUpload,
						Name:           "Throughput Upload ACON",
						Enabled:        true,
						Endpoint:       "/api/v1/speedtest/upload",
						SizeMB:         2,
						TimeoutSeconds: 45,
					},
					{
						ID:                   "switch_discovery",
						Type:                 TaskTypeDiscoveryProtocol,
						Name:                 "Descoberta de Switch LLDP/CDP",
						Enabled:              true,
						Protocol:             "LLDP_CDP",
						ListenTimeoutSeconds: 45,
						TimeoutSeconds:       60,
					},
				},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}
