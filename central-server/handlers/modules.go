package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"central-server/config"
	"central-server/db"
	"central-server/models"
)

// ModulesHandler gerencia o catálogo de módulos de teste e templates JSON
type ModulesHandler struct {
	cfg         *config.AppConfig
	mongoClient *db.MongoDBClient
	modulesDir  string
	mu          sync.RWMutex
}

// NewModulesHandler cria e inicializa o manipulador de catálogo de módulos
func NewModulesHandler(cfg *config.AppConfig, mongoClient *db.MongoDBClient) *ModulesHandler {
	modulesDir := filepath.Join("config", "modules")
	if err := os.MkdirAll(modulesDir, 0755); err != nil {
		log.Printf("[MODULES][AVISO] Falha ao criar diretório %s: %v", modulesDir, err)
	}

	h := &ModulesHandler{
		cfg:         cfg,
		mongoClient: mongoClient,
		modulesDir:  modulesDir,
	}

	// Garante que existam módulos padrão cadastrados se o diretório estiver vazio
	h.ensureDefaultModules()

	return h
}

// ValidTaskTypes lista todos os tipos de tarefas suportados
var ValidTaskTypes = map[models.TaskType]bool{
	models.TaskTypePing:               true,
	models.TaskTypePacketLoss:         true,
	models.TaskTypeThroughputDownload: true,
	models.TaskTypeThroughputUpload:   true,
	models.TaskTypeDNSLookup:          true,
	models.TaskTypeDiscoveryProtocol:  true,
	models.TaskTypeSystemInfo:         true,
	models.TaskTypeHTTPCheck:          true,
	models.TaskTypeTraceroute:         true,
}

// ensureDefaultModules cria arquivos padrão caso o catálogo esteja vazio
func (h *ModulesHandler) ensureDefaultModules() {
	h.mu.Lock()
	defer h.mu.Unlock()

	defaults := []models.TestTaskConfig{
		{
			ID:             "ping_gateway",
			Type:           models.TaskTypePing,
			Name:           "Latência Gateway Local",
			Enabled:        true,
			Target:         "auto_gateway",
			Count:          3,
			TimeoutMS:      1000,
			TimeoutSeconds: 15,
		},
		{
			ID:             "ping_nti_core",
			Type:           models.TaskTypePing,
			Name:           "Latência Core NTI (Anel)",
			Enabled:        true,
			Target:         "177.101.22.40",
			Count:          3,
			TimeoutMS:      1000,
			TimeoutSeconds: 15,
		},
		{
			ID:             "packet_loss_gateway",
			Type:           models.TaskTypePacketLoss,
			Name:           "Perda de Pacotes Gateway",
			Enabled:        true,
			Target:         "auto_gateway",
			Count:          10,
			TimeoutMS:      1000,
			TimeoutSeconds: 20,
		},
		{
			ID:             "dns_uepg",
			Type:           models.TaskTypeDNSLookup,
			Name:           "Resolução DNS Institucional",
			Enabled:        true,
			Domain:         "uepg.br",
			TimeoutMS:      2000,
			TimeoutSeconds: 10,
		},
		{
			ID:             "throughput_download",
			Type:           models.TaskTypeThroughputDownload,
			Name:           "Throughput Download ACON",
			Enabled:        true,
			Endpoint:       "/api/v1/speedtest/download",
			SizeMB:         5,
			TimeoutSeconds: 45,
		},
		{
			ID:             "throughput_upload",
			Type:           models.TaskTypeThroughputUpload,
			Name:           "Throughput Upload ACON",
			Enabled:        true,
			Endpoint:       "/api/v1/speedtest/upload",
			SizeMB:         2,
			TimeoutSeconds: 45,
		},
		{
			ID:                   "switch_discovery",
			Type:                 models.TaskTypeDiscoveryProtocol,
			Name:                 "Descoberta de Switch LLDP/CDP",
			Enabled:              true,
			Protocol:             "LLDP_CDP",
			ListenTimeoutSeconds: 45,
			TimeoutSeconds:       60,
		},
		{
			ID:             "http_check_portal",
			Type:           models.TaskTypeHTTPCheck,
			Name:           "Disponibilidade Portal UEPG",
			Enabled:        true,
			Endpoint:       "https://www.uepg.br",
			TimeoutSeconds: 15,
		},
	}

	for _, mod := range defaults {
		filePath := filepath.Join(h.modulesDir, mod.ID+".json")
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			data, errMarshal := json.MarshalIndent(mod, "", "  ")
			if errMarshal == nil {
				_ = os.WriteFile(filePath, data, 0644)
			}
		}
	}
}

// HandleModules gerencia listagem, criação/upload e exclusão de módulos
func (h *ModulesHandler) HandleModules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 1. GET /api/v1/admin/modules -> Lista módulos
	if r.Method == http.MethodGet {
		h.mu.RLock()
		defer h.mu.RUnlock()

		modules := make([]models.TestTaskConfig, 0)
		files, err := os.ReadDir(h.modulesDir)
		if err == nil {
			for _, file := range files {
				if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".json") {
					content, errRead := os.ReadFile(filepath.Join(h.modulesDir, file.Name()))
					if errRead == nil {
						var mod models.TestTaskConfig
						if errJSON := json.Unmarshal(content, &mod); errJSON == nil && mod.ID != "" {
							modules = append(modules, mod)
						}
					}
				}
			}
		}

		if len(modules) == 0 {
			h.mu.RUnlock()
			h.ensureDefaultModules()
			h.mu.RLock()

			files, err = os.ReadDir(h.modulesDir)
			if err == nil {
				for _, file := range files {
					if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".json") {
						content, errRead := os.ReadFile(filepath.Join(h.modulesDir, file.Name()))
						if errRead == nil {
							var mod models.TestTaskConfig
							if errJSON := json.Unmarshal(content, &mod); errJSON == nil && mod.ID != "" {
								modules = append(modules, mod)
							}
						}
					}
				}
			}
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(modules)
		return
	}

	// 2. POST /api/v1/admin/modules -> Cria / Faz Upload de novo módulo
	if r.Method == http.MethodPost {
		var mod models.TestTaskConfig
		if err := json.NewDecoder(r.Body).Decode(&mod); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error":   "payload JSON inválido",
				"details": err.Error(),
			})
			return
		}

		mod.ID = strings.TrimSpace(mod.ID)
		mod.Name = strings.TrimSpace(mod.Name)

		if mod.ID == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "o campo 'id' do módulo é obrigatório"})
			return
		}
		if mod.Name == "" {
			mod.Name = mod.ID
		}

		// Valida se o tipo de tarefa é suportado pela sonda
		if !ValidTaskTypes[mod.Type] {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": fmt.Sprintf("tipo de tarefa '%s' não suportado pela sonda Go", mod.Type),
				"supported_types": []string{
					string(models.TaskTypePing),
					string(models.TaskTypePacketLoss),
					string(models.TaskTypeThroughputDownload),
					string(models.TaskTypeThroughputUpload),
					string(models.TaskTypeDNSLookup),
					string(models.TaskTypeDiscoveryProtocol),
					string(models.TaskTypeHTTPCheck),
					string(models.TaskTypeTraceroute),
				},
			})
			return
		}

		// Garante defaults mínimos sensatos
		if mod.TimeoutSeconds <= 0 {
			mod.TimeoutSeconds = 15
		}

		h.mu.Lock()
		data, errMarshal := json.MarshalIndent(mod, "", "  ")
		if errMarshal != nil {
			h.mu.Unlock()
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "falha ao serializar módulo"})
			return
		}

		filePath := filepath.Join(h.modulesDir, mod.ID+".json")
		if errWrite := os.WriteFile(filePath, data, 0644); errWrite != nil {
			h.mu.Unlock()
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "falha ao salvar arquivo no disco"})
			return
		}
		h.mu.Unlock()

		log.Printf("[MODULES] Módulo '%s' (%s - %s) cadastrado com sucesso", mod.ID, mod.Name, mod.Type)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "módulo salvo com sucesso no catálogo",
			"module":  mod,
		})
		return
	}

	// 3. DELETE /api/v1/admin/modules/{id}
	if r.Method == http.MethodDelete {
		prefix := "/api/v1/admin/modules/"
		moduleID := strings.TrimPrefix(r.URL.Path, prefix)
		moduleID = strings.TrimSpace(moduleID)
		moduleID = strings.TrimSuffix(moduleID, ".json")

		if moduleID == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "id do módulo obrigatório na URL"})
			return
		}

		h.mu.Lock()
		filePath := filepath.Join(h.modulesDir, moduleID+".json")
		errRemove := os.Remove(filePath)
		h.mu.Unlock()

		if errRemove != nil && !os.IsNotExist(errRemove) {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "falha ao remover módulo"})
			return
		}

		log.Printf("[MODULES] Módulo '%s' removido do catálogo", moduleID)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "módulo removido com sucesso", "id": moduleID})
		return
	}

	http.Error(w, `{"error": "método não permitido"}`, http.StatusMethodNotAllowed)
}
