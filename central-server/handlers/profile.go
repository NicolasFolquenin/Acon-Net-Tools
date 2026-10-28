package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"central-server/config"
	"central-server/models"
)

// ProfileHandler gerencia os perfis dinâmicos de medição das sondas
type ProfileHandler struct {
	cfg         *config.AppConfig
	profilePath string
	profile     *models.MeasurementProfile
	mu          sync.RWMutex
}

// NewProfileHandler inicializa o gerenciador de perfis de medição
func NewProfileHandler(cfg *config.AppConfig) *ProfileHandler {
	profilePath := filepath.Join("config", "default_profile.json")

	h := &ProfileHandler{
		cfg:         cfg,
		profilePath: profilePath,
	}

	// Carrega perfil inicial do disco
	if err := h.reloadProfileFromDisk(); err != nil {
		log.Printf("[PROFILE][AVISO] Falha ao carregar perfil inicial de %s: %v. Usando perfil embutido.", profilePath, err)
		h.profile = h.getBuiltinProfile()
	}

	return h
}

// reloadProfileFromDisk lê o arquivo default_profile.json
func (h *ProfileHandler) reloadProfileFromDisk() error {
	data, err := os.ReadFile(h.profilePath)
	if err != nil {
		return err
	}

	var prof models.MeasurementProfile
	if err := json.Unmarshal(data, &prof); err != nil {
		return err
	}

	h.mu.Lock()
	h.profile = &prof
	h.mu.Unlock()

	log.Printf("[PROFILE] Perfil de medição '%s' carregado com sucesso (%d testes configurados).", prof.Version, len(prof.Tests))
	return nil
}

// getBuiltinProfile retorna um perfil de medição seguro caso o arquivo json não esteja acessível
func (h *ProfileHandler) getBuiltinProfile() *models.MeasurementProfile {
	return &models.MeasurementProfile{
		Version:         "2026.1-fallback",
		UpdatedAt:       time.Now().UTC(),
		IntervalMinutes: 30,
		SyncSchedule:    "fixed_half_hour",
		Description:     "Perfil padrão de fallback UEPG",
		Tests: []models.TestTaskConfig{
			{
				ID:        "ping_gateway",
				Type:      models.TaskTypePing,
				Name:      "Latência Gateway Local",
				Enabled:   true,
				Target:    "auto_gateway",
				Count:     3,
				TimeoutMS: 1000,
			},
			{
				ID:        "ping_nti_core",
				Type:      models.TaskTypePing,
				Name:      "Latência Core NTI (Anel)",
				Enabled:   true,
				Target:    "177.101.22.40",
				Count:     3,
				TimeoutMS: 1000,
			},
			{
				ID:        "dns_uepg",
				Type:      models.TaskTypeDNSLookup,
				Name:      "Resolução DNS Institucional",
				Enabled:   true,
				Domain:    "uepg.br",
				TimeoutMS: 2000,
			},
			{
				ID:             "throughput_download",
				Type:           models.TaskTypeThroughputDownload,
				Name:           "Throughput Download",
				Enabled:        true,
				Endpoint:       "/api/v1/speedtest/download",
				SizeMB:         5,
				TimeoutSeconds: 15,
			},
			{
				ID:             "throughput_upload",
				Type:           models.TaskTypeThroughputUpload,
				Name:           "Throughput Upload",
				Enabled:        true,
				Endpoint:       "/api/v1/speedtest/upload",
				SizeMB:         2,
				TimeoutSeconds: 15,
			},
			{
				ID:                   "network_switch_discovery",
				Type:                 models.TaskTypeDiscoveryProtocol,
				Name:                 "Identificação Switch / Porta",
				Enabled:              true,
				Protocol:             "LLDP_CDP",
				ListenTimeoutSeconds: 45,
			},
			{
				ID:      "system_hardware_info",
				Type:    models.TaskTypeSystemInfo,
				Name:    "Telemetria de Host e Interface",
				Enabled: true,
			},
		},
	}
}

// GetProfile entrega o manifesto de tarefas em JSON para as sondas
// Rota: GET /api/v1/probes/profile
func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "metodo nao permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	h.mu.RLock()
	prof := h.profile
	h.mu.RUnlock()

	if prof == nil {
		prof = h.getBuiltinProfile()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(prof)
}

// UpdateProfile permite atualizar o perfil dinâmico de medições
// Rota: PUT /api/v1/probes/profile
func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, `{"error": "metodo nao permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	var newProfile models.MeasurementProfile
	if err := json.NewDecoder(r.Body).Decode(&newProfile); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "formato JSON invalido", "details": err.Error()})
		return
	}

	if newProfile.Version == "" {
		newProfile.Version = time.Now().Format("2006.01.02.150405")
	}
	newProfile.UpdatedAt = time.Now().UTC()

	// Persiste em disco se possível
	data, err := json.MarshalIndent(newProfile, "", "  ")
	if err == nil {
		_ = os.WriteFile(h.profilePath, data, 0644)
	}

	h.mu.Lock()
	h.profile = &newProfile
	h.mu.Unlock()

	log.Printf("[PROFILE] Perfil atualizado para versao %s (%d testes configurados)", newProfile.Version, len(newProfile.Tests))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(newProfile)
}
