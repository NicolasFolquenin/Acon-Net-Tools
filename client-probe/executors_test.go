package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// --- Helpers de teste ---

// newTestExecCtx cria um ExecutionContext para testes com um servidor HTTP mockado
func newTestExecCtx(serverURL string) *ExecutionContext {
	return &ExecutionContext{
		ServerURL: serverURL,
		SysInfo: &SystemInfo{
			Hostname:       "TEST-PC-01",
			IPAddress:      "10.0.0.100",
			MACAddress:     "00:11:22:33:44:55",
			Gateway:        "10.0.0.1",
			NetworkAdapter: "Test Adapter",
		},
		KnownSwitch: nil,
		HTTPClient:  &http.Client{Timeout: 5 * time.Second},
	}
}

// --- Testes do ExecutePingTask ---

func TestExecutePingTask_AutoGateway(t *testing.T) {
	execCtx := newTestExecCtx("")
	task := TestTaskConfig{
		ID:        "ping_gateway",
		Type:      TaskTypePing,
		Name:      "Test Ping Gateway",
		Enabled:   true,
		Target:    "auto_gateway",
		Count:     1,
		TimeoutMS: 2000,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := ExecutePingTask(ctx, task, execCtx)

	if result.Unit != "ms" {
		t.Errorf("esperado unit='ms', obteve '%s'", result.Unit)
	}
	if result.Status != "ok" {
		t.Errorf("esperado status='ok', obteve '%s'", result.Status)
	}
	if result.Value <= 0 {
		t.Errorf("esperado valor > 0, obteve %.2f", result.Value)
	}
}

func TestExecutePingTask_FixedTarget(t *testing.T) {
	execCtx := newTestExecCtx("")
	task := TestTaskConfig{
		ID:        "ping_nti",
		Type:      TaskTypePing,
		Name:      "Test Ping Fixed IP",
		Enabled:   true,
		Target:    "1.1.1.1",
		Count:     1,
		TimeoutMS: 2000,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := ExecutePingTask(ctx, task, execCtx)

	if result.Unit != "ms" {
		t.Errorf("esperado unit='ms', obteve '%s'", result.Unit)
	}
	// Deve retornar ok (via nativo, TCP ou estimativa)
	if result.Status != "ok" {
		t.Errorf("esperado status='ok', obteve '%s'", result.Status)
	}
}

func TestExecutePingTask_DefaultsApplied(t *testing.T) {
	execCtx := newTestExecCtx("")
	execCtx.SysInfo = nil // sem gateway conhecido
	task := TestTaskConfig{
		ID:      "ping_defaults",
		Type:    TaskTypePing,
		Name:    "Test Ping Defaults",
		Enabled: true,
		Target:  "", // vazio → deve cair no fallback
		// Count e TimeoutMS em 0 → devem usar defaults
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := ExecutePingTask(ctx, task, execCtx)

	if result.Status != "ok" {
		t.Errorf("esperado status='ok' com defaults aplicados, obteve '%s'", result.Status)
	}
}

// --- Testes do ExecuteDNSLookupTask ---

func TestExecuteDNSLookupTask_ValidDomain(t *testing.T) {
	execCtx := newTestExecCtx("")
	task := TestTaskConfig{
		ID:        "dns_test",
		Type:      TaskTypeDNSLookup,
		Name:      "Test DNS Lookup",
		Enabled:   true,
		Domain:    "google.com",
		TimeoutMS: 3000,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := ExecuteDNSLookupTask(ctx, task, execCtx)

	if result.Unit != "ms" {
		t.Errorf("esperado unit='ms', obteve '%s'", result.Unit)
	}
	if result.Status != "ok" {
		t.Errorf("esperado status='ok', obteve '%s' (erro: %s)", result.Status, result.Error)
	}
	if result.Value <= 0 {
		t.Errorf("esperado latência DNS > 0 ms, obteve %.2f", result.Value)
	}
}

func TestExecuteDNSLookupTask_DefaultDomain(t *testing.T) {
	execCtx := newTestExecCtx("")
	task := TestTaskConfig{
		ID:      "dns_default",
		Type:    TaskTypeDNSLookup,
		Name:    "Test DNS Default",
		Enabled: true,
		Domain:  "", // vazio → deve usar "uepg.br"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := ExecuteDNSLookupTask(ctx, task, execCtx)

	// Pode falhar se uepg.br não resolver fora da rede UEPG, mas o comportamento deve ser coerente
	if result.Unit != "ms" {
		t.Errorf("esperado unit='ms', obteve '%s'", result.Unit)
	}
	if result.Status != "ok" && result.Status != "error" {
		t.Errorf("esperado status='ok' ou 'error', obteve '%s'", result.Status)
	}
}

func TestExecuteDNSLookupTask_InvalidDomain(t *testing.T) {
	execCtx := newTestExecCtx("")
	task := TestTaskConfig{
		ID:        "dns_invalid",
		Type:      TaskTypeDNSLookup,
		Name:      "Test DNS Invalid",
		Enabled:   true,
		Domain:    "dominio.inexistente.zzz",
		TimeoutMS: 2000,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := ExecuteDNSLookupTask(ctx, task, execCtx)

	if result.Status != "error" {
		t.Errorf("esperado status='error' para domínio inexistente, obteve '%s'", result.Status)
	}
	if result.Error == "" {
		t.Error("esperado mensagem de erro preenchida para domínio inexistente")
	}
}

// --- Testes do ExecuteThroughputDownloadTask ---

func TestExecuteThroughputDownloadTask_WithMockServer(t *testing.T) {
	// Cria servidor mock que retorna 1MB de dados
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, 1*1024*1024) // 1MB
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}))
	defer mockServer.Close()

	execCtx := newTestExecCtx(mockServer.URL)
	task := TestTaskConfig{
		ID:             "download_test",
		Type:           TaskTypeThroughputDownload,
		Name:           "Test Download",
		Enabled:        true,
		Endpoint:       "/download",
		SizeMB:         1,
		TimeoutSeconds: 10,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result := ExecuteThroughputDownloadTask(ctx, task, execCtx)

	if result.Unit != "Mbps" {
		t.Errorf("esperado unit='Mbps', obteve '%s'", result.Unit)
	}
	if result.Status != "ok" {
		t.Errorf("esperado status='ok', obteve '%s'", result.Status)
	}
	if result.Value <= 0 {
		t.Errorf("esperado throughput > 0 Mbps, obteve %.2f", result.Value)
	}
}

func TestExecuteThroughputDownloadTask_ServerOffline(t *testing.T) {
	// Servidor inexistente → deve cair no fallback de estimativa
	execCtx := newTestExecCtx("http://127.0.0.1:1")
	task := TestTaskConfig{
		ID:             "download_offline",
		Type:           TaskTypeThroughputDownload,
		Name:           "Test Download Offline",
		Enabled:        true,
		Endpoint:       "/api/v1/speedtest/download",
		SizeMB:         5,
		TimeoutSeconds: 2,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := ExecuteThroughputDownloadTask(ctx, task, execCtx)

	if result.Status != "ok" {
		t.Errorf("esperado status='ok' (fallback estimado), obteve '%s'", result.Status)
	}
	if result.Value <= 0 {
		t.Errorf("esperado valor estimado > 0, obteve %.2f", result.Value)
	}
}

// --- Testes do ExecuteThroughputUploadTask ---

func TestExecuteThroughputUploadTask_WithMockServer(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer mockServer.Close()

	execCtx := newTestExecCtx(mockServer.URL)
	task := TestTaskConfig{
		ID:             "upload_test",
		Type:           TaskTypeThroughputUpload,
		Name:           "Test Upload",
		Enabled:        true,
		Endpoint:       "/upload",
		SizeMB:         1,
		TimeoutSeconds: 10,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result := ExecuteThroughputUploadTask(ctx, task, execCtx)

	if result.Unit != "Mbps" {
		t.Errorf("esperado unit='Mbps', obteve '%s'", result.Unit)
	}
	if result.Status != "ok" {
		t.Errorf("esperado status='ok', obteve '%s'", result.Status)
	}
}

// --- Testes do ExecuteSystemInfoTask ---

func TestExecuteSystemInfoTask(t *testing.T) {
	execCtx := newTestExecCtx("")
	task := TestTaskConfig{
		ID:      "sysinfo_test",
		Type:    TaskTypeSystemInfo,
		Name:    "Test SysInfo",
		Enabled: true,
	}

	ctx := context.Background()
	sys, result := ExecuteSystemInfoTask(ctx, task, execCtx)

	if result.Status != "ok" {
		t.Errorf("esperado status='ok', obteve '%s'", result.Status)
	}
	if result.Unit != "host" {
		t.Errorf("esperado unit='host', obteve '%s'", result.Unit)
	}
	if sys == nil {
		t.Fatal("esperado SystemInfo não nil")
	}
	if sys.Hostname == "" {
		t.Error("esperado hostname preenchido")
	}
}

// --- Testes do ExecuteDiscoveryProtocolTask ---

func TestExecuteDiscoveryProtocolTask_WithCachedSwitch(t *testing.T) {
	execCtx := newTestExecCtx("")
	execCtx.KnownSwitch = &SwitchInfo{
		SwitchName:  "SW-BLOCO-L-P1",
		SwitchPort:  "GigabitEthernet1/0/5",
		VLAN:        120,
		SwitchModel: "Cisco Catalyst 2960",
		Protocol:    "LLDP",
	}

	task := TestTaskConfig{
		ID:                   "lldp_test",
		Type:                 TaskTypeDiscoveryProtocol,
		Name:                 "Test LLDP",
		Enabled:              true,
		Protocol:             "LLDP_CDP",
		ListenTimeoutSeconds: 2, // Timeout curto para teste
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sw, result := ExecuteDiscoveryProtocolTask(ctx, task, execCtx)

	if result.Status != "ok" {
		t.Errorf("esperado status='ok' com switch em cache, obteve '%s'", result.Status)
	}
	if sw == nil {
		t.Fatal("esperado SwitchInfo não nil quando há cache")
	}
	if sw.SwitchName != "SW-BLOCO-L-P1" {
		t.Errorf("esperado switch 'SW-BLOCO-L-P1', obteve '%s'", sw.SwitchName)
	}
}

// --- Testes da interface TaskExecutor e do Registry ---

func TestNewExecutorRegistry_AllTypesRegistered(t *testing.T) {
	registry := NewExecutorRegistry()

	expectedTypes := []TaskType{
		TaskTypePing,
		TaskTypeThroughputDownload,
		TaskTypeThroughputUpload,
		TaskTypeDNSLookup,
		TaskTypeDiscoveryProtocol,
		TaskTypeSystemInfo,
	}

	for _, tt := range expectedTypes {
		if _, exists := registry[tt]; !exists {
			t.Errorf("tipo de tarefa '%s' não está registrado no ExecutorRegistry", tt)
		}
	}
}

func TestTaskExecutorFunc_ImplementsInterface(t *testing.T) {
	// Verifica que TaskExecutorFunc satisfaz a interface TaskExecutor em tempo de compilação
	var _ TaskExecutor = TaskExecutorFunc(func(ctx context.Context, task TestTaskConfig, execCtx *ExecutionContext) TaskExecutionResult {
		return TaskExecutionResult{
			Metric: DynamicMetricResult{Value: 42, Unit: "test", Status: "ok"},
		}
	})
}

func TestRegisterExecutor_CustomType(t *testing.T) {
	probe := NewNetworkProbe()

	// Registra um executor customizado para um tipo novo
	customType := TaskType("custom_traceroute")
	probe.RegisterExecutor(customType, TaskExecutorFunc(func(ctx context.Context, task TestTaskConfig, execCtx *ExecutionContext) TaskExecutionResult {
		return TaskExecutionResult{
			Metric: DynamicMetricResult{
				Value:   5,
				Unit:    "hops",
				Status:  "ok",
				Details: fmt.Sprintf("Traceroute para %s", task.Target),
			},
		}
	}))

	// Verifica que o executor foi registrado
	if _, exists := probe.executors[customType]; !exists {
		t.Error("executor customizado não foi registrado")
	}
}

func TestRunProfile_WithRegistry(t *testing.T) {
	probe := NewNetworkProbe()

	profile := &MeasurementProfile{
		Version:         "test-1.0",
		IntervalMinutes: 30,
		Tests: []TestTaskConfig{
			{
				ID:        "ping_gateway",
				Type:      TaskTypePing,
				Name:      "Test Ping",
				Enabled:   true,
				Target:    "auto_gateway",
				Count:     1,
				TimeoutMS: 1000,
			},
			{
				ID:      "sysinfo",
				Type:    TaskTypeSystemInfo,
				Name:    "Test SysInfo",
				Enabled: true,
			},
			{
				ID:      "disabled_test",
				Type:    TaskTypeDNSLookup,
				Name:    "Should Be Skipped",
				Enabled: false, // Deve ser ignorado
			},
		},
	}

	cfg := Config{
		NodeID:   "TEST-NODE",
		Location: "Lab Teste",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	payload, err := probe.RunProfile(ctx, profile, cfg, nil, nil)
	if err != nil {
		t.Fatalf("RunProfile retornou erro: %v", err)
	}

	if payload == nil {
		t.Fatal("payload não deveria ser nil")
	}
	if payload.NodeID != "TEST-NODE" {
		t.Errorf("esperado NodeID='TEST-NODE', obteve '%s'", payload.NodeID)
	}
	if payload.ProfileVersion != "test-1.0" {
		t.Errorf("esperado ProfileVersion='test-1.0', obteve '%s'", payload.ProfileVersion)
	}

	// Verifica que as métricas habilitadas foram executadas
	if _, exists := payload.Metrics["ping_gateway"]; !exists {
		t.Error("métrica 'ping_gateway' deveria existir no payload")
	}
	if _, exists := payload.Metrics["sysinfo"]; !exists {
		t.Error("métrica 'sysinfo' deveria existir no payload")
	}
	// Verifica que a métrica desabilitada NÃO foi executada
	if _, exists := payload.Metrics["disabled_test"]; exists {
		t.Error("métrica 'disabled_test' NÃO deveria existir (está disabled)")
	}

	// Verifica SystemInfo preenchido
	if payload.SystemInfo == nil {
		t.Error("SystemInfo deveria estar preenchido após ExecuteSystemInfoTask")
	}
}

func TestExecutePacketLossTask(t *testing.T) {
	execCtx := newTestExecCtx("")
	task := TestTaskConfig{
		ID:        "packet_loss_test",
		Type:      TaskTypePacketLoss,
		Name:      "Test Packet Loss",
		Enabled:   true,
		Target:    "127.0.0.1",
		Count:     2,
		TimeoutMS: 500,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := ExecutePacketLossTask(ctx, task, execCtx)
	if result.Unit != "%" {
		t.Errorf("esperado unit='%%', obteve '%s'", result.Unit)
	}
	if result.Value < 0 || result.Value > 100 {
		t.Errorf("esperado 0 <= value <= 100, obteve %.2f", result.Value)
	}
}

