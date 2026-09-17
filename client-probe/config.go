package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

var (
	configMu  sync.RWMutex
	appConfig *Config
)

const defaultConfigFilename = "config.json"
const defaultEnvFilename = ".env"

// LoadEnvFile lê arquivos .env e carrega variáveis para o ambiente
func LoadEnvFile(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		return
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
			if os.Getenv(key) == "" {
				_ = os.Setenv(key, val)
			}
		}
	}
}

// LoadConfig carrega as configurações com a seguinte prioridade:
// 1. Variáveis de ambiente (ou arquivo .env)
// 2. Arquivo config.json (se existir)
// 3. Valores padrão
func LoadConfig(customPath string) (*Config, error) {
	configMu.Lock()
	defer configMu.Unlock()

	// Tenta carregar .env local e no diretório do executável
	LoadEnvFile(defaultEnvFilename)
	execDir, errExec := filepath.Abs(filepath.Dir(os.Args[0]))
	if errExec == nil {
		LoadEnvFile(filepath.Join(execDir, defaultEnvFilename))
	}

	targetPath := defaultConfigFilename
	if customPath != "" {
		targetPath = customPath
	} else if errExec == nil {
		execConfig := filepath.Join(execDir, defaultConfigFilename)
		if _, err := os.Stat(execConfig); err == nil {
			targetPath = execConfig
		}
	}

	cfg := &Config{
		IntervalMinutes: 60,
		PingTarget:      "1.1.1.1",
	}

	// 1. Tenta carregar do config.json se ele existir
	if data, err := os.ReadFile(targetPath); err == nil {
		_ = json.Unmarshal(data, cfg)
	}

	// 2. Sobrescreve com variáveis do .env / Ambiente (têm prioridade)
	if v := os.Getenv("PROBE_NODE_ID"); v != "" {
		cfg.NodeID = v
	}
	if v := os.Getenv("PROBE_LOCATION"); v != "" {
		cfg.Location = v
	}
	if v := os.Getenv("PROBE_SERVER_URL"); v != "" {
		cfg.ServerURL = v
	}
	if v := os.Getenv("PROBE_API_TOKEN"); v != "" {
		cfg.APIToken = v
	}
	if v := os.Getenv("PROBE_PING_TARGET"); v != "" {
		cfg.PingTarget = v
	}
	if v := os.Getenv("PROBE_INTERVAL_MINUTES"); v != "" {
		if intVal, err := strconv.Atoi(v); err == nil && intVal > 0 {
			cfg.IntervalMinutes = intVal
		}
	}

	// Fallbacks finais caso nem config.json nem .env tenham preenchido
	if cfg.NodeID == "" {
		if hostname, _ := os.Hostname(); hostname != "" {
			cfg.NodeID = hostname
		} else {
			cfg.NodeID = "meu-pc-windows"
		}
	}
	if cfg.Location == "" {
		cfg.Location = "Escritorio-Local"
	}
	if cfg.ServerURL == "" {
		cfg.ServerURL = "http://localhost:8080"
	}
	if cfg.APIToken == "" {
		cfg.APIToken = "super-secret-probe-token-2026"
	}
	if cfg.IntervalMinutes <= 0 {
		cfg.IntervalMinutes = 60
	}
	if cfg.PingTarget == "" {
		cfg.PingTarget = "1.1.1.1"
	}

	// Salva em config.json para manter o estado persistido e acessível pela UI
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err == nil {
		_ = os.WriteFile(targetPath, data, 0644)
	}

	appConfig = cfg
	log.Printf("[CONFIG] Configurações ativas: NodeID=%s, Location=%s, Server=%s, Interval=%dm, Ping=%s",
		cfg.NodeID, cfg.Location, cfg.ServerURL, cfg.IntervalMinutes, cfg.PingTarget)

	return appConfig, nil
}

// GetCurrentConfig retorna a configuração atualmente carregada em memória
func GetCurrentConfig() Config {
	configMu.RLock()
	defer configMu.RUnlock()
	if appConfig == nil {
		return Config{
			NodeID:          "meu-pc-windows",
			Location:        "Escritorio-Local",
			ServerURL:       "http://localhost:8080",
			APIToken:        "super-secret-probe-token-2026",
			IntervalMinutes: 60,
			PingTarget:      "1.1.1.1",
		}
	}
	return *appConfig
}

// SaveConfig atualiza o arquivo config.json e o arquivo .env
func SaveConfig(cfg Config) error {
	configMu.Lock()
	defer configMu.Unlock()

	// 1. Atualiza config.json
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	executableDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	targetJsonPath := filepath.Join(executableDir, defaultConfigFilename)
	if err := os.WriteFile(defaultConfigFilename, data, 0644); err != nil {
		_ = os.WriteFile(targetJsonPath, data, 0644)
	}

	// 2. Atualiza arquivo .env com os novos valores
	envContent := strings.Join([]string{
		"# ==============================================================================",
		"# CONFIGURAÇÕES DO CLIENT PROBE (WINDOWS)",
		"# ==============================================================================",
		"PROBE_NODE_ID=" + cfg.NodeID,
		"PROBE_LOCATION=" + cfg.Location,
		"PROBE_SERVER_URL=" + cfg.ServerURL,
		"PROBE_API_TOKEN=" + cfg.APIToken,
		"PROBE_INTERVAL_MINUTES=" + strconv.Itoa(cfg.IntervalMinutes),
		"PROBE_PING_TARGET=" + cfg.PingTarget,
		"",
	}, "\n")

	targetEnvPath := filepath.Join(executableDir, defaultEnvFilename)
	if err := os.WriteFile(defaultEnvFilename, []byte(envContent), 0644); err != nil {
		_ = os.WriteFile(targetEnvPath, []byte(envContent), 0644)
	}

	appConfig = &cfg
	log.Printf("[CONFIG] Configurações e .env atualizados com sucesso")
	return nil
}
