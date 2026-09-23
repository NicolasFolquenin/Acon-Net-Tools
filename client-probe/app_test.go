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
