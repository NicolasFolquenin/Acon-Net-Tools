package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"central-server/config"
	"central-server/db"
	"central-server/models"
)

// GroupsHandler gerencia as requisições de grupos de teste e envio de diretrizes às sondas
type GroupsHandler struct {
	db  *db.MongoDBClient
	cfg *config.AppConfig
}

// NewGroupsHandler cria uma nova instância de GroupsHandler
func NewGroupsHandler(database *db.MongoDBClient, cfg *config.AppConfig) *GroupsHandler {
	return &GroupsHandler{
		db:  database,
		cfg: cfg,
	}
}

// GetProbeConfig trata GET /api/v1/config/probe?vlan={vlan}&hostname={hostname}
// Chamado pela sonda (client-probe) para descobrir quais módulos de testes executar
func (h *GroupsHandler) GetProbeConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "metodo nao permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	vlanQuery := r.URL.Query().Get("vlan")
	hostname := strings.TrimSpace(r.URL.Query().Get("hostname"))

	vlan := 0
	if vlanQuery != "" {
		if parsed, err := strconv.Atoi(vlanQuery); err == nil && parsed > 0 {
			vlan = parsed
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	group, err := h.db.GetGroupByVLAN(ctx, vlan)
	if err != nil {
		log.Printf("[GROUPS][ERRO] Falha ao obter grupo para VLAN %d (host %s): %v", vlan, hostname, err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "erro ao recuperar diretrizes de testes",
		})
		return
	}

	resp := models.ProbeConfigResponse{
		GroupID:         group.ID,
		GroupName:       group.Name,
		MatchedVLAN:     vlan,
		IsDefault:       group.IsDefault,
		IntervalMinutes: group.IntervalMinutes,
		Modules:         group.Modules,
	}

	log.Printf("[GROUPS] Config entregue para host '%s' (VLAN %d): Grupo '%s' (Padrao: %v)",
		hostname, vlan, group.Name, group.IsDefault)

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// HandleGroups multiplexa requisições REST para /api/v1/groups
func (h *GroupsHandler) HandleGroups(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	// Rota: /api/v1/groups
	if len(parts) == 3 {
		switch r.Method {
		case http.MethodGet:
			h.listGroups(w, r)
		case http.MethodPost:
			h.createGroup(w, r)
		default:
			http.Error(w, `{"error": "metodo nao permitido"}`, http.StatusMethodNotAllowed)
		}
		return
	}

	// Rota: /api/v1/groups/{id}
	if len(parts) == 4 {
		groupID := parts[3]
		switch r.Method {
		case http.MethodGet:
			h.getGroup(w, r, groupID)
		case http.MethodPut:
			h.updateGroup(w, r, groupID)
		case http.MethodDelete:
			h.deleteGroup(w, r, groupID)
		default:
			http.Error(w, `{"error": "metodo nao permitido"}`, http.StatusMethodNotAllowed)
		}
		return
	}

	http.NotFound(w, r)
}

func (h *GroupsHandler) listGroups(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	groups, err := h.db.ListGroups(ctx)
	if err != nil {
		log.Printf("[GROUPS][ERRO] Falha ao listar grupos: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "falha ao listar grupos"})
		return
	}

	_ = json.NewEncoder(w).Encode(groups)
}

func (h *GroupsHandler) getGroup(w http.ResponseWriter, r *http.Request, groupID string) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	group, err := h.db.GetGroupByID(ctx, groupID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "grupo nao encontrado"})
		return
	}

	_ = json.NewEncoder(w).Encode(group)
}

func (h *GroupsHandler) createGroup(w http.ResponseWriter, r *http.Request) {
	var group models.TestGroup
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "payload invalido: " + err.Error()})
		return
	}

	if strings.TrimSpace(group.Name) == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "nome do grupo e obrigatorio"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	newID, err := h.db.CreateGroup(ctx, &group)
	if err != nil {
		log.Printf("[GROUPS][ERRO] Falha ao criar grupo: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	group.ID = newID
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(group)
}

func (h *GroupsHandler) updateGroup(w http.ResponseWriter, r *http.Request, groupID string) {
	var group models.TestGroup
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "payload invalido: " + err.Error()})
		return
	}

	if strings.TrimSpace(group.Name) == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "nome do grupo e obrigatorio"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.db.UpdateGroup(ctx, groupID, &group); err != nil {
		log.Printf("[GROUPS][ERRO] Falha ao atualizar grupo %s: %v", groupID, err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	group.ID = groupID
	_ = json.NewEncoder(w).Encode(group)
}

func (h *GroupsHandler) deleteGroup(w http.ResponseWriter, r *http.Request, groupID string) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.db.DeleteGroup(ctx, groupID); err != nil {
		log.Printf("[GROUPS][ERRO] Falha ao excluir grupo %s: %v", groupID, err)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{"status": "grupo excluido com sucesso"})
}
