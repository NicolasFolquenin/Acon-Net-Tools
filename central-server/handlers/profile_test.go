package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"central-server/config"
	"central-server/models"
)

func TestProfileHandlerGetAndPut(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "profile_test_*.json")
	if err != nil {
		t.Fatalf("falha ao criar arquivo temp: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	cfg := &config.AppConfig{
		APIToken: "test-token",
	}

	h := &ProfileHandler{
		cfg:         cfg,
		profilePath: tmpFile.Name(),
	}

	// 1. Testa fallback inicial
	req := httptest.NewRequest(http.MethodGet, "/api/v1/probes/profile", nil)
	w := httptest.NewRecorder()
	h.GetProfile(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("esperado status 200 em GetProfile, recebido %d", w.Code)
	}

	var prof models.MeasurementProfile
	if err := json.NewDecoder(w.Body).Decode(&prof); err != nil {
		t.Fatalf("erro ao decodificar JSON do perfil: %v", err)
	}
	if len(prof.Tests) == 0 {
		t.Fatal("esperado testes configurados no perfil retornado")
	}

	// 2. Testa UpdateProfile (PUT)
	newProf := models.MeasurementProfile{
		Version:         "2026.test",
		IntervalMinutes: 15,
		Tests: []models.TestTaskConfig{
			{
				ID:      "ping_custom",
				Type:    models.TaskTypePing,
				Name:    "Ping Customizado",
				Enabled: true,
				Target:  "8.8.8.8",
			},
		},
	}

	putBytes, _ := json.Marshal(newProf)
	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/probes/profile", bytes.NewReader(putBytes))
	putW := httptest.NewRecorder()
	h.UpdateProfile(putW, putReq)

	if putW.Code != http.StatusOK {
		t.Fatalf("esperado status 200 em UpdateProfile, recebido %d", putW.Code)
	}

	// 3. Testa se GetProfile reflete a atualização
	getReq2 := httptest.NewRequest(http.MethodGet, "/api/v1/probes/profile", nil)
	getW2 := httptest.NewRecorder()
	h.GetProfile(getW2, getReq2)

	var updatedProf models.MeasurementProfile
	_ = json.NewDecoder(getW2.Body).Decode(&updatedProf)
	if updatedProf.Version != "2026.test" || len(updatedProf.Tests) != 1 {
		t.Fatalf("perfil não foi atualizado corretamente: %+v", updatedProf)
	}
}
