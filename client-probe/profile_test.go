package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestProfileManagerFallbackAndCache(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "profile_mgr_test_*.json")
	if err != nil {
		t.Fatalf("falha ao criar arquivo temp: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	pm := &ProfileManager{
		cachePath: tmpFile.Name(),
	}

	fb := pm.GetBuiltinFallback()
	if fb == nil || len(fb.Tests) == 0 {
		t.Fatal("esperado perfil de fallback com testes configurados")
	}

	// Salva em disco e testa recarregamento
	pm.saveToDisk(fb)

	loaded, err := pm.loadFromDisk()
	if err != nil {
		t.Fatalf("erro ao recarregar perfil do disco: %v", err)
	}
	if loaded.Version != fb.Version || len(loaded.Tests) != len(fb.Tests) {
		t.Fatalf("perfil recarregado diverge do original: %+v", loaded)
	}
}

func TestRunProfileModular(t *testing.T) {
	probe := NewNetworkProbe()

	customProfile := &MeasurementProfile{
		Version:         "2026.test-run",
		IntervalMinutes: 30,
		Tests: []TestTaskConfig{
			{
				ID:        "ping_gateway",
				Type:      TaskTypePing,
				Name:      "Ping Teste",
				Enabled:   true,
				Target:    "127.0.0.1",
				Count:     1,
				TimeoutMS: 500,
			},
			{
				ID:        "dns_test",
				Type:      TaskTypeDNSLookup,
				Name:      "DNS Teste",
				Enabled:   true,
				Domain:    "localhost",
				TimeoutMS: 1000,
			},
			{
				ID:      "disabled_test",
				Type:    TaskTypePing,
				Name:    "Teste Desabilitado",
				Enabled: false,
				Target:  "8.8.8.8",
			},
		},
	}

	cfg := Config{
		NodeID:    "TEST-NODE",
		Location:  "Laboratorio-Teste",
		ServerURL: "http://localhost:8090",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	payload, err := probe.RunProfile(ctx, customProfile, cfg, nil, nil)
	if err != nil {
		t.Fatalf("erro ao executar RunProfile: %v", err)
	}

	if payload.NodeID != "TEST-NODE" {
		t.Fatalf("esperado NodeID TEST-NODE, obtido %s", payload.NodeID)
	}
	if payload.ProfileVersion != "2026.test-run" {
		t.Fatalf("esperado ProfileVersion 2026.test-run, obtido %s", payload.ProfileVersion)
	}
	if payload.Metrics == nil {
		t.Fatal("esperado mapa de metricas preenchido")
	}
	if _, ok := payload.Metrics["ping_gateway"]; !ok {
		t.Fatal("esperado resultado de ping_gateway nas metricas")
	}
	if _, ok := payload.Metrics["dns_test"]; !ok {
		t.Fatal("esperado resultado de dns_test nas metricas")
	}
	if _, ok := payload.Metrics["disabled_test"]; ok {
		t.Fatal("teste desabilitado nao deveria estar presente no mapa de metricas")
	}
}

func TestFetchRemoteProfileForHost_200Success(t *testing.T) {
	expectedProfile := MeasurementProfile{
		Version:         "2026.host-test",
		IntervalMinutes: 30,
		Tests: []TestTaskConfig{
			{
				ID:      "ping_test",
				Type:    TaskTypePing,
				Name:    "Ping Host Test",
				Enabled: true,
				Target:  "127.0.0.1",
			},
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/probes/profile/TEST-HOST-01" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(expectedProfile)
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	pm := NewProfileManager()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	prof, err := pm.FetchRemoteProfileForHost(ctx, ts.URL, "test-token", HostBootstrapInfo{Hostname: "TEST-HOST-01"})
	if err != nil {
		t.Fatalf("FetchRemoteProfileForHost falhou: %v", err)
	}
	if prof == nil || prof.Version != "2026.host-test" {
		t.Fatalf("esperado versao 2026.host-test, obteve %+v", prof)
	}
}

func TestFetchRemoteProfileForHost_404AutoRegister(t *testing.T) {
	registeredProfile := MeasurementProfile{
		Version:         "2026.registered-bootstrap",
		IntervalMinutes: 30,
		Tests: []TestTaskConfig{
			{
				ID:      "ping_central",
				Type:    TaskTypePing,
				Name:    "Ping Central Server",
				Enabled: true,
			},
		},
	}

	registeredCalled := false

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/probes/profile/NEW-HOST-99" {
			// Simula máquina nova: 404 Not Found
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error": "machine not found"}`))
			return
		}
		if r.URL.Path == "/api/v1/probes/register" && r.Method == http.MethodPost {
			registeredCalled = true
			var regReq ProbeRegisterRequest
			_ = json.NewDecoder(r.Body).Decode(&regReq)
			if regReq.Hostname != "NEW-HOST-99" {
				t.Errorf("esperado hostname NEW-HOST-99 no registro, obteve %s", regReq.Hostname)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(registeredProfile)
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	pm := NewProfileManager()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	hostInfo := HostBootstrapInfo{
		Hostname:   "NEW-HOST-99",
		IPAddress:  "10.0.0.99",
		MACAddress: "11:22:33:44:55:66",
		VLAN:       10,
	}

	prof, err := pm.FetchRemoteProfileForHost(ctx, ts.URL, "test-token", hostInfo)
	if err != nil {
		t.Fatalf("FetchRemoteProfileForHost falhou: %v", err)
	}
	if !registeredCalled {
		t.Error("esperava que endpoint /api/v1/probes/register fosse chamado após 404")
	}
	if prof == nil || prof.Version != "2026.registered-bootstrap" {
		t.Fatalf("esperado perfil registrado com versao 2026.registered-bootstrap, obteve %+v", prof)
	}
}
