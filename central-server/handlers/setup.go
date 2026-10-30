package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"central-server/config"
	"central-server/db"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

// SetupHandler gerencia o assistente de primeiro acesso (Web Wizard)
type SetupHandler struct {
	cfg         *config.AppConfig
	mongoClient *db.MongoDBClient
}

// NewSetupHandler instancia o manipulador do setup wizard
func NewSetupHandler(cfg *config.AppConfig, mongoClient *db.MongoDBClient) *SetupHandler {
	return &SetupHandler{
		cfg:         cfg,
		mongoClient: mongoClient,
	}
}

// SetupStatusResponse resposta de status do assistente
type SetupStatusResponse struct {
	SetupCompleted  bool                   `json:"setup_completed"`
	DetectedIP      string                 `json:"detected_ip"`
	CurrentSettings map[string]interface{} `json:"current_settings"`
}

// SetupCompleteRequest payload enviado pelo formulário do wizard
type SetupCompleteRequest struct {
	Port              string `json:"port"`
	PublicServerURL   string `json:"public_server_url"`
	DNSServerTarget   string `json:"dns_server_target"`
	DNSDomainTarget   string `json:"dns_domain_target"`
	DNSServerName     string `json:"dns_server_name"`
	PingCoreTarget    string `json:"ping_core_target"`
	PingCoreName      string `json:"ping_core_name"`
	DefaultPingTarget string `json:"default_ping_target"`
	AdminUser         string `json:"admin_user"`
	AdminPassword     string `json:"admin_password"`
	APIToken          string `json:"api_token"`
}

// HandleStatus verifica se o setup já foi concluído e entrega dados iniciais
func (h *SetupHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "metodo nao permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	detectedIP := detectLocalIP()

	resp := SetupStatusResponse{
		SetupCompleted: h.cfg.SetupCompleted,
		DetectedIP:     detectedIP,
		CurrentSettings: map[string]interface{}{
			"port":                h.cfg.Port,
			"public_server_url":   h.cfg.PublicServerURL,
			"dns_server_target":   h.cfg.DNSServerTarget,
			"dns_domain_target":   h.cfg.DNSDomainTarget,
			"dns_server_name":     h.cfg.DNSServerName,
			"ping_core_target":    h.cfg.PingCoreTarget,
			"ping_core_name":      h.cfg.PingCoreName,
			"default_ping_target": h.cfg.DefaultPingTarget,
			"admin_user":          h.cfg.AdminUser,
		},
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// HandleComplete valida as respostas, grava no .env e inicializa o sistema
func (h *SetupHandler) HandleComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "metodo nao permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Trava de segurança: impede reconfiguração se já foi finalizado
	if h.cfg.SetupCompleted {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "O assistente de primeiro acesso já foi concluído anteriormente.",
		})
		return
	}

	var req SetupCompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "JSON inválido: " + err.Error(),
		})
		return
	}

	// Normalizações e fallbacks
	port := strings.TrimSpace(req.Port)
	if port == "" {
		port = "8085"
	}

	dnsServer := strings.TrimSpace(req.DNSServerTarget)
	if dnsServer == "" {
		dnsServer = "1.1.1.1"
	}

	dnsDomain := strings.TrimSpace(req.DNSDomainTarget)
	if dnsDomain == "" {
		dnsDomain = "google.com"
	}

	dnsName := strings.TrimSpace(req.DNSServerName)
	if dnsName == "" {
		dnsName = "DNS Primário"
	}

	pingCore := strings.TrimSpace(req.PingCoreTarget)
	if pingCore == "" {
		pingCore = "1.1.1.1"
	}

	pingCoreName := strings.TrimSpace(req.PingCoreName)
	if pingCoreName == "" {
		pingCoreName = "Roteador Central (Core)"
	}

	defaultPing := strings.TrimSpace(req.DefaultPingTarget)
	if defaultPing == "" {
		defaultPing = "auto_gateway"
	}

	publicURL := strings.TrimSpace(req.PublicServerURL)
	if publicURL == "" {
		detectedIP := detectLocalIP()
		publicURL = fmt.Sprintf("http://%s:%s", detectedIP, port)
	}
	if !strings.HasPrefix(publicURL, "http://") && !strings.HasPrefix(publicURL, "https://") {
		publicURL = "http://" + publicURL
	}
	publicURL = strings.TrimSuffix(publicURL, "/")

	adminUser := strings.TrimSpace(req.AdminUser)
	if adminUser == "" {
		adminUser = "admin"
	}

	adminPass := strings.TrimSpace(req.AdminPassword)
	if len(adminPass) < 4 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "A senha do administrador deve ter pelo menos 4 caracteres.",
		})
		return
	}

	apiToken := strings.TrimSpace(req.APIToken)
	if apiToken == "" {
		apiToken = "super-secret-probe-token-2026"
	}

	updates := map[string]string{
		"PORT":                port,
		"PUBLIC_SERVER_URL":   publicURL,
		"DNS_SERVER_TARGET":   dnsServer,
		"DNS_DOMAIN_TARGET":   dnsDomain,
		"DNS_SERVER_NAME":     dnsName,
		"PING_CORE_TARGET":    pingCore,
		"PING_CORE_NAME":      pingCoreName,
		"DEFAULT_PING_TARGET": defaultPing,
		"AUTH_ADMIN_USER":     adminUser,
		"AUTH_ADMIN_PASSWORD": adminPass,
		"API_TOKEN":           apiToken,
	}

	// 1. Salva no .env e atualiza memória
	if err := h.cfg.MarkSetupCompleted(updates); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "Falha ao gravar arquivo .env: " + err.Error(),
		})
		return
	}

	// 2. Atualiza ou cria usuário administrador no MongoDB
	if h.mongoClient != nil && h.mongoClient.UsersCollection != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		hash, err := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.DefaultCost)
		if err == nil {
			now := time.Now().UTC()
			filter := bson.M{"username": strings.ToLower(adminUser)}
			update := bson.M{
				"$set": bson.M{
					"password_hash": string(hash),
					"role":          "admin",
					"auth_type":     "local",
					"updated_at":    now,
				},
				"$setOnInsert": bson.M{
					"created_at": now,
				},
			}
			opts := options.Update().SetUpsert(true)
			_, _ = h.mongoClient.UsersCollection.UpdateOne(ctx, filter, update, opts)
		}
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "success",
		"message":    "Configuração inicial concluída com sucesso! Redirecionando para o login...",
		"public_url": publicURL,
	})
}

// detectLocalIP busca a interface e IP local principal
func detectLocalIP() string {
	conn, err := net.Dial("udp", "1.1.1.1:80")
	if err == nil {
		defer conn.Close()
		localAddr := conn.LocalAddr().(*net.UDPAddr)
		return localAddr.IP.String()
	}
	return "127.0.0.1"
}
