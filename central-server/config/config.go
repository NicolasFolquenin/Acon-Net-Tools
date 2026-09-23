package config

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
)

// AppConfig armazena as configurações do Servidor Central
type AppConfig struct {
	Port            string
	MongoURI        string
	DatabaseName    string
	CollectionName  string
	APIToken        string
	StaticDir       string
	QueryHoursLimit int
	AdminUser       string
	AdminPass       string
	ViewerUser      string
	ViewerPass      string
	SessionSecret   string
	OIDCEnabled      bool
	OIDCIssuerURL    string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCRedirectURL  string
	OIDCAdminGroup   string
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

	cfg := &AppConfig{
		Port:             getEnv("PORT", "8080"),
		MongoURI:         getEnv("MONGO_URI", "mongodb://localhost:27017"),
		DatabaseName:     getEnv("MONGO_DB", "network_central"),
		CollectionName:   getEnv("MONGO_COLLECTION", "metrics"),
		APIToken:         getEnv("API_TOKEN", "super-secret-probe-token-2026"),
		StaticDir:        staticDir,
		QueryHoursLimit:  getEnvAsInt("QUERY_HOURS_LIMIT", 24),
		AdminUser:        getEnv("AUTH_ADMIN_USER", "admin"),
		AdminPass:        getEnv("AUTH_ADMIN_PASSWORD", "admin123"),
		ViewerUser:       getEnv("AUTH_VIEWER_USER", "operador"),
		ViewerPass:       getEnv("AUTH_VIEWER_PASSWORD", "operador123"),
		SessionSecret:    getEnv("SESSION_SECRET", "super-secret-session-key-change-in-prod-2026"),
		OIDCEnabled:      getEnvAsBool("OIDC_ENABLED", false),
		OIDCIssuerURL:    strings.TrimSuffix(getEnv("OIDC_ISSUER_URL", ""), "/"),
		OIDCClientID:     getEnv("OIDC_CLIENT_ID", ""),
		OIDCClientSecret: getEnv("OIDC_CLIENT_SECRET", ""),
		OIDCRedirectURL:  getEnv("OIDC_REDIRECT_URL", ""),
		OIDCAdminGroup:   getEnv("OIDC_ADMIN_GROUP", "noc-admins"),
	}

	log.Printf("[CONFIG] Servidor configurado: Port=%s, MongoURI=%s, DB=%s, Collection=%s, OIDC=%t",
		cfg.Port, cfg.MongoURI, cfg.DatabaseName, cfg.CollectionName, cfg.OIDCEnabled)
	return cfg
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
