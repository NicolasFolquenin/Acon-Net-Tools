package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"central-server/config"
	"central-server/models"
)

func setupTestMachineHandler(t *testing.T) (*MachineProfileHandler, string) {
	tempDir, err := os.MkdirTemp("", "machines-test-*")
	if err != nil {
		t.Fatalf("falha ao criar temp dir: %v", err)
	}

	cfg := &config.AppConfig{
		APIToken: "test-token-123",
	}

	h := &MachineProfileHandler{
		cfg:         cfg,
		mongoClient: nil, // Testes em memória/disco sem necessidade de MongoDB ativo
		machinesDir: tempDir,
	}

	return h, tempDir
}

func TestMachineProfileHandler_404ForNewMachine(t *testing.T) {
	h, tempDir := setupTestMachineHandler(t)
	defer os.RemoveAll(tempDir)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/probes/profile/UNKNOWN-HOST", nil)
	rr := httptest.NewRecorder()

	h.HandleGetProfileByHostname(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("esperado HTTP 404 para maquina nova, obteve %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("falha ao decodificar resposta JSON: %v", err)
	}

	if resp["error"] != "machine not found" {
		t.Errorf("esperado error='machine not found', obteve '%v'", resp["error"])
	}
}

func TestMachineProfileHandler_RegisterWithVLANReitoria(t *testing.T) {
	h, tempDir := setupTestMachineHandler(t)
	defer os.RemoveAll(tempDir)

	registerBody := models.ProbeRegisterRequest{
		Hostname:   "REITORIA-GAB-01",
		IPAddress:  "10.10.10.45",
		MACAddress: "00:1A:2B:3C:4D:5E",
		VLAN:       10, // VLAN 10 deve casar com Grupo Reitoria
		SwitchName: "SW-REITORIA-01",
	}
	bodyData, _ := json.Marshal(registerBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/probes/register", bytes.NewReader(bodyData))
	rr := httptest.NewRecorder()

	h.HandleRegisterProbe(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("esperado HTTP 201 Created no registro, obteve %d: %s", rr.Code, rr.Body.String())
	}

	var prof models.MeasurementProfile
	if err := json.Unmarshal(rr.Body.Bytes(), &prof); err != nil {
		t.Fatalf("falha ao decodificar profile criado: %v", err)
	}

	// Verifica se gerou o arquivo no disco
	expectedFile := filepath.Join(tempDir, "REITORIA-GAB-01.json")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("esperado que arquivo %s fosse criado no disco", expectedFile)
	}

	// Deve ter testes do grupo reitoria (ex: packet_loss_core)
	hasPacketLoss := false
	for _, task := range prof.Tests {
		if task.Type == models.TaskTypePacketLoss {
			hasPacketLoss = true
			break
		}
	}
	if !hasPacketLoss {
		t.Errorf("esperado teste de packet_loss no perfil da Reitoria")
	}

	// Agora, consulta subsequente deve retornar 200 OK!
	reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/probes/profile/REITORIA-GAB-01", nil)
	rrGet := httptest.NewRecorder()
	h.HandleGetProfileByHostname(rrGet, reqGet)

	if rrGet.Code != http.StatusOK {
		t.Errorf("esperado HTTP 200 na consulta da maquina registrada, obteve %d", rrGet.Code)
	}
}

func TestMachineProfileHandler_RegisterDefaultGroup(t *testing.T) {
	h, tempDir := setupTestMachineHandler(t)
	defer os.RemoveAll(tempDir)

	registerBody := models.ProbeRegisterRequest{
		Hostname:   "MAQUINA-DESCONHECIDA-01",
		IPAddress:  "192.168.1.100",
		MACAddress: "AA:BB:CC:DD:EE:FF",
		VLAN:       999, // VLAN sem regra cadastrada
	}
	bodyData, _ := json.Marshal(registerBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/probes/register", bytes.NewReader(bodyData))
	rr := httptest.NewRecorder()

	h.HandleRegisterProbe(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("esperado HTTP 201 Created, obteve %d", rr.Code)
	}

	var prof models.MeasurementProfile
	_ = json.Unmarshal(rr.Body.Bytes(), &prof)

	// Grupo padrão deve conter o ciclo completo de medição (ping gateway, download, upload e dns)
	if len(prof.Tests) != 1 {
		t.Errorf("esperado grupo default com testes completos (exatamente 1 teste), obteve %d testes", len(prof.Tests))
	}
}

func TestMachineProfileHandler_AdminListGroups(t *testing.T) {
	h, tempDir := setupTestMachineHandler(t)
	defer os.RemoveAll(tempDir)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/groups", nil)
	rr := httptest.NewRecorder()

	h.HandleAdminGroups(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("esperado HTTP 200, obteve %d", rr.Code)
	}

	var groups []models.MachineGroup
	if err := json.Unmarshal(rr.Body.Bytes(), &groups); err != nil {
		t.Fatalf("falha ao decodificar grupos: %v", err)
	}

	if len(groups) < 3 {
		t.Errorf("esperado pelo menos 3 grupos iniciais, obteve %d", len(groups))
	}
}
