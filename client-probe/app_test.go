package main

import (
	"testing"
	"time"
)

func TestGetNextSyncSlot(t *testing.T) {
	loc := time.UTC

	tests := []struct {
		name     string
		input    time.Time
		expected time.Time
	}{
		{
			name:     "Minute before 30 (13:14:25) -> should be 13:30:00",
			input:    time.Date(2026, 9, 23, 13, 14, 25, 0, loc),
			expected: time.Date(2026, 9, 23, 13, 30, 0, 0, loc),
		},
		{
			name:     "Minute at 00 (13:00:05) -> should be 13:30:00",
			input:    time.Date(2026, 9, 23, 13, 0, 5, 0, loc),
			expected: time.Date(2026, 9, 23, 13, 30, 0, 0, loc),
		},
		{
			name:     "Minute after 30 (13:35:10) -> should be 14:00:00",
			input:    time.Date(2026, 9, 23, 13, 35, 10, 0, loc),
			expected: time.Date(2026, 9, 23, 14, 0, 0, 0, loc),
		},
		{
			name:     "Minute at 30 (13:30:00) -> should be 14:00:00",
			input:    time.Date(2026, 9, 23, 13, 30, 0, 0, loc),
			expected: time.Date(2026, 9, 23, 14, 0, 0, 0, loc),
		},
		{
			name:     "Minute at 59 (13:59:59) -> should be 14:00:00",
			input:    time.Date(2026, 9, 23, 13, 59, 59, 0, loc),
			expected: time.Date(2026, 9, 23, 14, 0, 0, 0, loc),
		},
		{
			name:     "Day/Year transition (23:45 on Dec 31) -> should be 00:00 on Jan 01 next year",
			input:    time.Date(2026, 12, 31, 23, 45, 0, 0, loc),
			expected: time.Date(2027, 1, 1, 0, 0, 0, 0, loc),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getNextSyncSlot(tt.input)
			if !got.Equal(tt.expected) {
				t.Errorf("getNextSyncSlot(%v) = %v; expected %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsCentralSynced(t *testing.T) {
	cfgEnabled := &Config{
		ServerURL:       "https://net-tools.acon.dev.uepg.br",
		SyncWithCentral: true,
	}
	if !cfgEnabled.IsCentralSynced() {
		t.Errorf("Expected IsCentralSynced to be true when ServerURL is valid and SyncWithCentral is true")
	}

	cfgDisabled := &Config{
		ServerURL:       "https://net-tools.acon.dev.uepg.br",
		SyncWithCentral: false,
	}
	if cfgDisabled.IsCentralSynced() {
		t.Errorf("Expected IsCentralSynced to be false when SyncWithCentral is false")
	}

	cfgEmptyURL := &Config{
		ServerURL:       "",
		SyncWithCentral: true,
	}
	if cfgEmptyURL.IsCentralSynced() {
		t.Errorf("Expected IsCentralSynced to be false when ServerURL is empty")
	}
}

func TestDefaultModulesConfig(t *testing.T) {
	mods := DefaultModulesConfig()
	if !mods.Ping.Enabled {
		t.Errorf("Expected Ping to be enabled by default")
	}
	if !mods.PacketLoss.Enabled {
		t.Errorf("Expected PacketLoss to be enabled by default")
	}
	if !mods.Download.Enabled {
		t.Errorf("Expected Download to be enabled by default")
	}
	if !mods.Upload.Enabled {
		t.Errorf("Expected Upload to be enabled by default")
	}
	if !mods.DNS.Enabled {
		t.Errorf("Expected DNS to be enabled by default")
	}
	if !mods.HTTPCheck.Enabled {
		t.Errorf("Expected HTTPCheck to be enabled by default")
	}
	if !mods.LLDP.Enabled {
		t.Errorf("Expected LLDP to be enabled by default")
	}
}

func TestNodeIDLocked(t *testing.T) {
	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.NodeID == "" {
		t.Errorf("Expected NodeID to be populated")
	}

	// Tenta forçar alteração
	cfg.NodeID = "novo-nome-hackeado"
	_ = SaveConfig(*cfg)

	// Recarrega e valida se permaneceu travado no hostname
	reloaded, _ := LoadConfig("")
	if reloaded.NodeID == "novo-nome-hackeado" {
		t.Errorf("NodeID should not be modifiable by user")
	}
}

