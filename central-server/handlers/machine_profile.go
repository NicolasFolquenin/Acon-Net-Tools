package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"central-server/config"
	"central-server/db"
	"central-server/models"
)

// MachineProfileHandler gerencia os perfis individuais por máquina e grupos
type MachineProfileHandler struct {
	cfg         *config.AppConfig
	mongoClient *db.MongoDBClient
	machinesDir string
	mu          sync.RWMutex
}

// NewMachineProfileHandler cria e inicializa o gerenciador de perfis por máquina
func NewMachineProfileHandler(cfg *config.AppConfig, mongoClient *db.MongoDBClient) *MachineProfileHandler {
	machinesDir := filepath.Join("config", "machines")
	if err := os.MkdirAll(machinesDir, 0755); err != nil {
		log.Printf("[MACHINE-PROFILE][AVISO] Falha ao criar diretório %s: %v", machinesDir, err)
	}

	return &MachineProfileHandler{
		cfg:         cfg,
		mongoClient: mongoClient,
		machinesDir: machinesDir,
	}
}

// HandleGetProfileByHostname lida com requisições para /api/v1/probes/profile/{hostname}
// Se a máquina não tiver configuração -> Retorna HTTP 404
// Se a máquina tiver configuração -> Retorna HTTP 200 com seu MeasurementProfile
func (h *MachineProfileHandler) HandleGetProfileByHostname(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "metodo nao permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	prefix := "/api/v1/probes/profile/"
	hostname := strings.TrimPrefix(r.URL.Path, prefix)
	hostname = strings.TrimSpace(hostname)
	// Remove possível extensão .json caso o cliente passe
	hostname = strings.TrimSuffix(hostname, ".json")

	if hostname == "" {
		http.Error(w, `{"error": "hostname obrigatorio na URL"}`, http.StatusBadRequest)
		return
	}

	h.mu.RLock()
	filePath := filepath.Join(h.machinesDir, hostname+".json")
	data, errFile := os.ReadFile(filePath)
	h.mu.RUnlock()

	if errFile == nil {
		var prof models.MeasurementProfile
		if errJSON := json.Unmarshal(data, &prof); errJSON == nil {
			hasThroughput := false
			for _, t := range prof.Tests {
				if t.Type == models.TaskTypeThroughputDownload {
					hasThroughput = true
					break
				}
			}
			// Se já possui throughput ou é um perfil customizado, retorna diretamente
			if hasThroughput || len(prof.Tests) > 2 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(data)
				return
			}
		}
	}

	// Se não achou arquivo em disco, tenta buscar no MongoDB
	if h.mongoClient != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		mach, errDB := h.mongoClient.GetMachine(ctx, hostname)
		if errDB == nil && mach != nil {
			// Reconstrói e salva o arquivo
			prof := h.buildProfileForMachine(ctx, mach)
			_ = h.saveMachineFile(hostname, prof)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(prof)
			return
		}
	}

	// Não encontrado -> Retorna 404 para indicar que é máquina nova
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error":    "machine not found",
		"hostname": hostname,
		"message":  "Maquina nao possui configuracao salva. Realize o auto-registro em POST /api/v1/probes/register",
	})
}

// HandleRegisterProbe lida com o auto-registro de novas máquinas via POST /api/v1/probes/register
func (h *MachineProfileHandler) HandleRegisterProbe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "metodo nao permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	var req models.ProbeRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "payload JSON invalido", "details": err.Error()})
		return
	}

	req.Hostname = strings.TrimSpace(req.Hostname)
	if req.Hostname == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "campo hostname obrigatorio"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// 1. Determina grupo através de regras automáticas (VLAN e Sub-rede IP)
	assignedGroup := h.matchGroupForProbe(ctx, req.VLAN, req.IPAddress)

	// 2. Se a máquina já foi cadastrada anteriormente com outro hostname mas mesmo MAC físico, remove o registro antigo
	normMAC := normalizeMAC(req.MACAddress)
	if isValidMAC(normMAC) && h.mongoClient != nil {
		if existingList, errList := h.mongoClient.ListMachines(ctx); errList == nil {
			for _, em := range existingList {
				if normalizeMAC(em.MACAddress) == normMAC && !strings.EqualFold(em.Hostname, req.Hostname) {
					log.Printf("[MACHINE-PROFILE] Maquina fisica renomeada detectada: '%s' -> '%s' (MAC: %s). Excluindo registro obsoleto.",
						em.Hostname, req.Hostname, normMAC)
					_ = h.mongoClient.DeleteMachine(ctx, em.Hostname)
					_ = os.Remove(filepath.Join(h.machinesDir, em.Hostname+".json"))
				}
			}
		}
	}

	// 3. Cria registro da máquina
	now := time.Now().UTC()
	mach := &models.MachineRegistration{
		Hostname:   req.Hostname,
		IPAddress:  req.IPAddress,
		MACAddress: req.MACAddress,
		VLAN:       req.VLAN,
		SwitchName: req.SwitchName,
		SwitchIP:   req.SwitchIP,
		SwitchPort: req.SwitchPort,
		GroupID:    assignedGroup.ID,
		IsCustom:   false,
		Status:     "active",
		LastSeen:   now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if h.mongoClient != nil {
		if err := h.mongoClient.SaveMachine(ctx, mach); err != nil {
			log.Printf("[MACHINE-PROFILE][AVISO] Falha ao persistir maquina %s no MongoDB: %v", mach.Hostname, err)
		}
	}

	// 4. Constrói o perfil correspondente e grava em disco como {hostname}.json
	prof := h.buildProfileForMachine(ctx, mach)
	if err := h.saveMachineFile(mach.Hostname, prof); err != nil {
		log.Printf("[MACHINE-PROFILE][ERRO] Falha ao gravar %s.json em disco: %v", mach.Hostname, err)
	}

	log.Printf("[MACHINE-PROFILE] Nova maquina registrada: %s (IP: %s, VLAN: %d) -> Grupo: %s",
		mach.Hostname, mach.IPAddress, mach.VLAN, mach.GroupID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(prof)
}

// HandleAdminMachines lida com a listagem e atualização de máquinas por administradores
func (h *MachineProfileHandler) HandleAdminMachines(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if r.Method == http.MethodGet {
		var machines []models.MachineRegistration
		if h.mongoClient != nil {
			var err error
			machines, err = h.mongoClient.ListMachines(ctx)
			if err != nil {
				machines = []models.MachineRegistration{}
			}

			// Mapa de hostnames e MACs já conhecidos
			knownHostnames := make(map[string]bool)
			knownMACs := make(map[string]string) // mac -> hostname mais recente

			for _, m := range machines {
				knownHostnames[strings.ToLower(m.Hostname)] = true
				mac := normalizeMAC(m.MACAddress)
				if isValidMAC(mac) {
					knownMACs[mac] = m.Hostname
				}
			}

			distinctNodes, errNodes := h.mongoClient.GetDistinctNodes(ctx)
			if errNodes == nil {
				for _, nodeID := range distinctNodes {
					trimmed := strings.TrimSpace(nodeID)
					if trimmed == "" || knownHostnames[strings.ToLower(trimmed)] {
						continue
					}

					// Busca a última métrica para inspecionar MAC
					latest, errLatest := h.mongoClient.GetLatestMetricForNode(ctx, trimmed)
					if errLatest != nil || latest == nil {
						continue
					}

					nodeMAC := ""
					if latest.SystemInfo != nil {
						nodeMAC = normalizeMAC(latest.SystemInfo.MACAddress)
					}

					// Se o MAC já pertence a outra máquina cadastrada, trata-se de um nó renomeado/antigo -> NÃO DUPLICA!
					if isValidMAC(nodeMAC) {
						if _, exists := knownMACs[nodeMAC]; exists {
							log.Printf("[MACHINE-PROFILE] No '%s' ignorado no cadastro pois MAC '%s' ja pertence a maquina ativa", trimmed, nodeMAC)
							continue
						}
					}

					knownHostnames[strings.ToLower(trimmed)] = true
					if isValidMAC(nodeMAC) {
						knownMACs[nodeMAC] = trimmed
					}

					synth := models.MachineRegistration{
						Hostname:  trimmed,
						GroupID:   "acon",
						IsCustom:  false,
						Status:    "active",
						CreatedAt: time.Now().UTC(),
						UpdatedAt: time.Now().UTC(),
						LastSeen:  latest.Timestamp,
					}
					if latest.SystemInfo != nil {
						synth.IPAddress = latest.SystemInfo.IPAddress
						synth.MACAddress = latest.SystemInfo.MACAddress
					}
					if latest.SwitchInfo != nil {
						synth.SwitchName = latest.SwitchInfo.SwitchName
						synth.SwitchPort = latest.SwitchInfo.SwitchPort
						synth.VLAN = latest.SwitchInfo.VLAN
					}
					if latest.VLAN > 0 && synth.VLAN == 0 {
						synth.VLAN = latest.VLAN
					}

					_ = h.mongoClient.SaveMachine(ctx, &synth)
					prof := h.buildProfileForMachine(ctx, &synth)
					_ = h.saveMachineFile(synth.Hostname, prof)

					machines = append(machines, synth)
				}
			}

			// Desduplicação estrita de 'machines' por MAC Address físico:
			// Se houver dois registros com o mesmo MAC, o mais recente permanece e o obsoleto é purgado
			macMap := make(map[string]models.MachineRegistration)
			var nonMacMachines []models.MachineRegistration
			var obsoleteHostnames []string

			for _, m := range machines {
				mac := normalizeMAC(m.MACAddress)
				if isValidMAC(mac) {
					if existing, found := macMap[mac]; !found {
						macMap[mac] = m
					} else {
						// Compara LastSeen para escolher a mais recente
						if m.LastSeen.After(existing.LastSeen) {
							obsoleteHostnames = append(obsoleteHostnames, existing.Hostname)
							macMap[mac] = m
						} else {
							obsoleteHostnames = append(obsoleteHostnames, m.Hostname)
						}
					}
				} else {
					nonMacMachines = append(nonMacMachines, m)
				}
			}

			for _, obsHost := range obsoleteHostnames {
				_ = h.mongoClient.DeleteMachine(ctx, obsHost)
				_ = os.Remove(filepath.Join(h.machinesDir, obsHost+".json"))
				log.Printf("[MACHINE-PROFILE] Excluida maquina obsoleta duplicada por MAC: %s", obsHost)
			}

			deduped := make([]models.MachineRegistration, 0, len(macMap)+len(nonMacMachines))
			for _, m := range macMap {
				deduped = append(deduped, m)
			}
			deduped = append(deduped, nonMacMachines...)

			// Ordena pelo último sinal mais recente/próximo primeiro
			sort.Slice(deduped, func(i, j int) bool {
				if !deduped[i].LastSeen.Equal(deduped[j].LastSeen) {
					return deduped[i].LastSeen.After(deduped[j].LastSeen)
				}
				return deduped[i].Hostname < deduped[j].Hostname
			})

			machines = deduped
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(machines)
		return
	}

	if r.Method == http.MethodPut {
		// PUT /api/v1/admin/machines/{hostname}
		prefix := "/api/v1/admin/machines/"
		hostname := strings.TrimPrefix(r.URL.Path, prefix)
		hostname = strings.TrimSpace(hostname)

		if hostname == "" {
			http.Error(w, `{"error": "hostname obrigatorio na URL"}`, http.StatusBadRequest)
			return
		}

		var updateReq models.UpdateMachineRequest
		if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "payload JSON invalido", "details": err.Error()})
			return
		}

		var mach *models.MachineRegistration
		if h.mongoClient != nil {
			if err := h.mongoClient.UpdateMachine(ctx, hostname, &updateReq); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			mach, _ = h.mongoClient.GetMachine(ctx, hostname)
		}

		if mach == nil {
			mach = &models.MachineRegistration{
				Hostname:    hostname,
				GroupID:     updateReq.GroupID,
				IsCustom:    updateReq.IsCustom,
				CustomTests: updateReq.CustomTests,
				Status:      updateReq.Status,
			}
		}

		// Atualiza o arquivo {hostname}.json
		prof := h.buildProfileForMachine(ctx, mach)
		_ = h.saveMachineFile(hostname, prof)

		log.Printf("[MACHINE-PROFILE] Maquina '%s' atualizada pelo admin: Grupo=%s, Custom=%t",
			hostname, mach.GroupID, mach.IsCustom)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "maquina atualizada com sucesso",
			"machine": mach,
			"profile": prof,
		})
		return
	}

	http.Error(w, `{"error": "metodo nao permitido"}`, http.StatusMethodNotAllowed)
}

// HandleAdminGroups lida com listagem e criação/atualização de grupos por administradores
func (h *MachineProfileHandler) HandleAdminGroups(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if r.Method == http.MethodGet {
		var groups []models.MachineGroup
		if h.mongoClient != nil {
			groups, _ = h.mongoClient.ListMachineGroups(ctx)
		}
		if len(groups) == 0 {
			groups = models.GetInitialGroups()
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(groups)
		return
	}

	if r.Method == http.MethodPost || r.Method == http.MethodPut {
		var grp models.MachineGroup
		if err := json.NewDecoder(r.Body).Decode(&grp); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "payload JSON invalido", "details": err.Error()})
			return
		}

		grp.ID = strings.TrimSpace(grp.ID)
		if grp.ID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "campo id do grupo obrigatorio"})
			return
		}

		if grp.CreatedAt.IsZero() {
			grp.CreatedAt = time.Now().UTC()
		}
		grp.UpdatedAt = time.Now().UTC()
		if grp.Profile.Version == "" {
			grp.Profile.Version = time.Now().Format("2006.01.02.150405")
		}

		if h.mongoClient != nil {
			if err := h.mongoClient.SaveMachineGroup(ctx, &grp); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
		}

		log.Printf("[MACHINE-PROFILE] Grupo '%s' (%s) salvo com sucesso.", grp.ID, grp.Name)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(grp)
		return
	}

	if r.Method == http.MethodDelete {
		prefix := "/api/v1/admin/groups/"
		groupID := strings.TrimPrefix(r.URL.Path, prefix)
		groupID = strings.TrimSpace(groupID)
		if groupID == "" || groupID == "default" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "o grupo padrão não pode ser excluído"})
			return
		}

		if h.mongoClient != nil {
			if err := h.mongoClient.DeleteMachineGroup(ctx, groupID); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
		}

		log.Printf("[MACHINE-PROFILE] Grupo '%s' excluído com sucesso.", groupID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "grupo excluído com sucesso"})
		return
	}

	http.Error(w, `{"error": "metodo nao permitido"}`, http.StatusMethodNotAllowed)
}

// --- Métodos Auxiliares ---

// matchGroupForProbe avalia regras automáticas de VLAN e IP para descobrir o grupo da máquina
func (h *MachineProfileHandler) matchGroupForProbe(ctx context.Context, vlan int, ipStr string) models.MachineGroup {
	groups := models.GetInitialGroups()
	if h.mongoClient != nil {
		if dbGroups, err := h.mongoClient.ListMachineGroups(ctx); err == nil && len(dbGroups) > 0 {
			groups = dbGroups
		}
	}

	var defaultGroup models.MachineGroup
	for _, grp := range groups {
		if grp.ID == "default" {
			defaultGroup = grp
		}

		// 1. Verifica match por VLAN
		if vlan > 0 {
			for _, v := range grp.Rules.VLANs {
				if v == vlan {
					log.Printf("[MACHINE-PROFILE] Regra correspondente por VLAN %d -> Grupo '%s'", vlan, grp.ID)
					return grp
				}
			}
		}

		// 2. Verifica match por Sub-rede IP
		if ipStr != "" {
			for _, subnet := range grp.Rules.IPSubnets {
				if matchSubnet(ipStr, subnet) {
					log.Printf("[MACHINE-PROFILE] Regra correspondente por IP %s em %s -> Grupo '%s'", ipStr, subnet, grp.ID)
					return grp
				}
			}
		}
	}

	if defaultGroup.ID != "" {
		return defaultGroup
	}
	return models.GetInitialGroups()[0]
}

// buildProfileForMachine compõe o MeasurementProfile final para uma máquina
func (h *MachineProfileHandler) buildProfileForMachine(ctx context.Context, mach *models.MachineRegistration) models.MeasurementProfile {
	now := time.Now().UTC()

	// Se a máquina possui testes customizados definidos pelo administrador
	if mach.IsCustom && len(mach.CustomTests) > 0 {
		return models.MeasurementProfile{
			Version:         fmt.Sprintf("custom-%s-%d", mach.Hostname, now.Unix()),
			UpdatedAt:       now,
			IntervalMinutes: 30,
			SyncSchedule:    "fixed_half_hour",
			Description:     fmt.Sprintf("Perfil customizado para a maquina %s", mach.Hostname),
			Tests:           mach.CustomTests,
		}
	}

	// Caso contrário, herda do grupo correspondente
	groups := models.GetInitialGroups()
	if h.mongoClient != nil {
		if grp, err := h.mongoClient.GetMachineGroup(ctx, mach.GroupID); err == nil && grp != nil {
			prof := grp.Profile
			prof.Version = fmt.Sprintf("%s-%s", grp.ID, mach.Hostname)
			prof.UpdatedAt = now
			return prof
		}
		if dbGroups, err := h.mongoClient.ListMachineGroups(ctx); err == nil && len(dbGroups) > 0 {
			groups = dbGroups
		}
	}

	for _, grp := range groups {
		if grp.ID == mach.GroupID {
			prof := grp.Profile
			prof.Version = fmt.Sprintf("%s-%s", grp.ID, mach.Hostname)
			prof.UpdatedAt = now
			return prof
		}
	}

	// Fallback para o primeiro grupo inicial (default)
	def := models.GetInitialGroups()[0]
	prof := def.Profile
	prof.Version = fmt.Sprintf("default-%s", mach.Hostname)
	prof.UpdatedAt = now
	return prof
}

// saveMachineFile grava o arquivo {hostname}.json na pasta config/machines/
func (h *MachineProfileHandler) saveMachineFile(hostname string, prof models.MeasurementProfile) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	data, err := json.MarshalIndent(prof, "", "  ")
	if err != nil {
		return err
	}

	filePath := filepath.Join(h.machinesDir, hostname+".json")
	return os.WriteFile(filePath, data, 0644)
}

// matchSubnet verifica se um endereço IP pertence a uma máscara CIDR (ex: 10.10.10.0/24)
func matchSubnet(ipStr, cidrStr string) bool {
	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return false
	}
	_, ipNet, err := net.ParseCIDR(strings.TrimSpace(cidrStr))
	if err != nil {
		return false
	}
	return ipNet.Contains(ip)
}
