package config

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// AppConfig armazena as configurações do Servidor Central
type AppConfig struct {
	Port                string `json:"port"`
	MongoURI            string `json:"-"`
	DatabaseName        string `json:"-"`
	CollectionName      string `json:"-"`
	APIToken            string `json:"-"`
	StaticDir           string `json:"-"`
	QueryHoursLimit     int    `json:"query_hours_limit"`
	AdminUser           string `json:"-"`
	AdminPass           string `json:"-"`
	ViewerUser          string `json:"-"`
	ViewerPass          string `json:"-"`
	SessionSecret       string `json:"-"`
	OIDCEnabled         bool   `json:"oidc_enabled"`
	OIDCIssuerURL       string `json:"oidc_issuer_url,omitempty"`
	OIDCClientID        string `json:"oidc_client_id,omitempty"`
	OIDCClientSecret    string `json:"-"`
	OIDCRedirectURL     string `json:"oidc_redirect_url,omitempty"`
	OIDCAdminGroup      string `json:"oidc_admin_group,omitempty"`
	PublicServerURL     string `json:"public_server_url"`
	DNSServerTarget     string `json:"dns_server_target"`
	DNSDomainTarget     string `json:"dns_domain_target"`
	DNSServerName       string `json:"dns_server_name"`
	PingCoreTarget      string `json:"ping_core_target"`
	PingCoreName        string `json:"ping_core_name"`
	DefaultPingTarget   string `json:"default_ping_target"`
	SpeedtestDownloadMB int    `json:"speedtest_download_mb"`
	SpeedtestUploadMB   int    `json:"speedtest_upload_mb"`
	SetupCompleted      bool   `json:"setup_completed"`
	mu                  sync.RWMutex
}

var currentConfig *AppConfig

// GetConfig retorna a instância ativa da configuração carregada ou carrega se nulo
func GetConfig() *AppConfig {
	if currentConfig == nil {
		return LoadConfig()
	}
	return currentConfig
}

// LoadEnvFile faz o parse de um arquivo .env simples (chave=valor) sem dependências externas
func LoadEnvFile(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		return // Arquivo .env opcional se as variáveis já vierem do SO/Docker
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, `"'`)
			// Só define se ainda não existir no ambiente
			if os.Getenv(key) == "" {
				_ = os.Setenv(key, val)
			}
		}
	}
}

// LoadConfig carrega as variáveis de ambiente com suporte a leitura de .env e fallbacks seguros
func LoadConfig() *AppConfig {
	// Tenta carregar .env local caso esteja rodando diretamente (ex: go run main.go)
	LoadEnvFile(".env")

	staticDir := getEnv("STATIC_DIR", "./public")
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		if _, err2 := os.Stat("./public"); err2 == nil {
			staticDir = "./public"
		}
	}

	setupCompleted := getEnvAsBool("SETUP_COMPLETED", false)
	// Checa também marcador de arquivo em disco
	setupMarker := filepath.Join("config", ".setup_done")
	if _, err := os.Stat(setupMarker); err == nil {
		setupCompleted = true
	}

	cfg := &AppConfig{
		Port:                getEnv("PORT", "8085"),
		MongoURI:            getEnv("MONGO_URI", "mongodb://localhost:27017"),
		DatabaseName:        getEnv("MONGO_DB", "network_central"),
		CollectionName:      getEnv("MONGO_COLLECTION", "metrics"),
		APIToken:            getEnv("API_TOKEN", "super-secret-probe-token-2026"),
		StaticDir:           staticDir,
		QueryHoursLimit:     getEnvAsInt("QUERY_HOURS_LIMIT", 24),
		AdminUser:           getEnv("AUTH_ADMIN_USER", "admin"),
		AdminPass:           getEnv("AUTH_ADMIN_PASSWORD", "admin123"),
		ViewerUser:          getEnv("AUTH_VIEWER_USER", "operador"),
		ViewerPass:          getEnv("AUTH_VIEWER_PASSWORD", "operador123"),
		SessionSecret:       getEnv("SESSION_SECRET", "super-secret-session-key-change-in-prod-2026"),
		OIDCEnabled:         getEnvAsBool("OIDC_ENABLED", false),
		OIDCIssuerURL:       strings.TrimSuffix(getEnv("OIDC_ISSUER_URL", ""), "/"),
		OIDCClientID:        getEnv("OIDC_CLIENT_ID", ""),
		OIDCClientSecret:    getEnv("OIDC_CLIENT_SECRET", ""),
		OIDCRedirectURL:     getEnv("OIDC_REDIRECT_URL", ""),
		OIDCAdminGroup:      getEnv("OIDC_ADMIN_GROUP", "noc-admins"),
		PublicServerURL:     strings.TrimSuffix(getEnv("PUBLIC_SERVER_URL", getEnv("SERVER_EXTERNAL_URL", "")), "/"),
		DNSServerTarget:     getEnv("DNS_SERVER_TARGET", "1.1.1.1"),
		DNSDomainTarget:     getEnv("DNS_DOMAIN_TARGET", "google.com"),
		DNSServerName:       getEnv("DNS_SERVER_NAME", "DNS Primário"),
		PingCoreTarget:      getEnv("PING_CORE_TARGET", "1.1.1.1"),
		PingCoreName:        getEnv("PING_CORE_NAME", "Roteador Central (Core)"),
		DefaultPingTarget:   getEnv("DEFAULT_PING_TARGET", "auto_gateway"),
		SpeedtestDownloadMB: getEnvAsInt("SPEEDTEST_DOWNLOAD_MB", 5),
		SpeedtestUploadMB:   getEnvAsInt("SPEEDTEST_UPLOAD_MB", 2),
		SetupCompleted:      setupCompleted,
	}

	currentConfig = cfg

	log.Printf("[CONFIG] Servidor configurado: Port=%s, SetupCompleted=%t, DNS_Server=%s, DNS_Domain=%s, Core_Target=%s",
		cfg.Port, cfg.SetupCompleted, cfg.DNSServerTarget, cfg.DNSDomainTarget, cfg.PingCoreTarget)
	return cfg
}

// UpdateEnvFile atualiza ou insere chaves no arquivo .env preservando comentários
func UpdateEnvFile(filename string, updates map[string]string) error {
	var lines []string
	found := make(map[string]bool)

	if data, err := os.ReadFile(filename); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			line := scanner.Text()
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
				parts := strings.SplitN(trimmed, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					if val, ok := updates[key]; ok {
						lines = append(lines, fmt.Sprintf("%s=%s", key, val))
						found[key] = true
						continue
					}
				}
			}
			lines = append(lines, line)
		}
	}

	for k, v := range updates {
		if !found[k] {
			lines = append(lines, fmt.Sprintf("%s=%s", k, v))
		}
	}

	content := strings.Join(lines, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return os.WriteFile(filename, []byte(content), 0644)
}

// MarkSetupCompleted salva as configurações no .env, atualiza a memória e cria o marcador permanente
func (cfg *AppConfig) MarkSetupCompleted(updates map[string]string) error {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	updates["SETUP_COMPLETED"] = "true"

	// Atualiza arquivo .env
	_ = UpdateEnvFile(".env", updates)

	// Atualiza variáveis no processo
	for k, v := range updates {
		_ = os.Setenv(k, v)
	}

	// Atualiza campos da struct em memória
	if v, ok := updates["PORT"]; ok && v != "" {
		cfg.Port = v
	}
	if v, ok := updates["PUBLIC_SERVER_URL"]; ok && v != "" {
		cfg.PublicServerURL = v
	}
	if v, ok := updates["DNS_SERVER_TARGET"]; ok && v != "" {
		cfg.DNSServerTarget = v
	}
	if v, ok := updates["DNS_DOMAIN_TARGET"]; ok && v != "" {
		cfg.DNSDomainTarget = v
	}
	if v, ok := updates["DNS_SERVER_NAME"]; ok && v != "" {
		cfg.DNSServerName = v
	}
	if v, ok := updates["PING_CORE_TARGET"]; ok && v != "" {
		cfg.PingCoreTarget = v
	}
	if v, ok := updates["PING_CORE_NAME"]; ok && v != "" {
		cfg.PingCoreName = v
	}
	if v, ok := updates["DEFAULT_PING_TARGET"]; ok && v != "" {
		cfg.DefaultPingTarget = v
	}
	if v, ok := updates["AUTH_ADMIN_USER"]; ok && v != "" {
		cfg.AdminUser = v
	}
	if v, ok := updates["AUTH_ADMIN_PASSWORD"]; ok && v != "" {
		cfg.AdminPass = v
	}
	if v, ok := updates["API_TOKEN"]; ok && v != "" {
		cfg.APIToken = v
	}

	cfg.SetupCompleted = true

	// Cria marcador permanente em disco
	_ = os.MkdirAll("config", 0755)
	setupMarker := filepath.Join("config", ".setup_done")
	_ = os.WriteFile(setupMarker, []byte("completed"), 0644)

	log.Printf("[SETUP] Assistente de primeiro acesso concluído com sucesso!")
	return nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getEnvAsBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		lower := strings.ToLower(strings.TrimSpace(val))
		return lower == "true" || lower == "1" || lower == "yes" || lower == "sim"
	}
	return defaultVal
}
