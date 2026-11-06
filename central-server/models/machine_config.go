package models

import "time"

// MachineRegistration representa os dados de uma máquina registrada no Central Server
type MachineRegistration struct {
	Hostname     string           `bson:"_id" json:"hostname"`             // Hostname é a chave primária única
	IPAddress    string           `bson:"ip_address" json:"ip_address"`
	MACAddress   string           `bson:"mac_address" json:"mac_address"`
	VLAN         int              `bson:"vlan" json:"vlan"`                 // VLAN reportada (via LLDP/CDP)
	SwitchName   string           `bson:"switch_name" json:"switch_name"`
	SwitchIP     string           `bson:"switch_ip" json:"switch_ip"`
	SwitchPort   string           `bson:"switch_port" json:"switch_port"`
	GroupID      string           `bson:"group_id" json:"group_id"`         // "default", "reitoria", "acon", etc.
	IsCustom     bool             `bson:"is_custom" json:"is_custom"`       // Se tem perfil customizado em vez do herdado do grupo
	CustomTests  []TestTaskConfig `bson:"custom_tests,omitempty" json:"custom_tests,omitempty"`
	Status       string            `bson:"status" json:"status"`             // "active", "pending_review"
	LastSeen     time.Time         `bson:"last_seen" json:"last_seen"`
	CreatedAt    time.Time         `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time         `bson:"updated_at" json:"updated_at"`
	SystemInfo   *SystemInfo       `bson:"system_info,omitempty" json:"system_info,omitempty"`
	LoggedUser   string            `bson:"logged_user,omitempty" json:"logged_user,omitempty"`
	ActiveUser   string            `bson:"active_user,omitempty" json:"active_user,omitempty"`
	Users        []UserAccountInfo `bson:"users,omitempty" json:"users,omitempty"`
}

// ProbeRegisterRequest representa o payload enviado pela sonda no primeiro contato (quando 404)
type ProbeRegisterRequest struct {
	Hostname   string `json:"hostname"`
	IPAddress  string `json:"ip_address"`
	MACAddress string `json:"mac_address"`
	VLAN       int    `json:"vlan"`
	SwitchName string `json:"switch_name"`
	SwitchIP   string `json:"switch_ip"`
	SwitchPort string `json:"switch_port"`
}

// UpdateMachineRequest representa a requisição do admin para alterar o grupo ou testes de uma máquina
type UpdateMachineRequest struct {
	GroupID     string           `json:"group_id"`
	IsCustom    bool             `json:"is_custom"`
	CustomTests []TestTaskConfig `json:"custom_tests,omitempty"`
	Status      string           `json:"status,omitempty"`
}
