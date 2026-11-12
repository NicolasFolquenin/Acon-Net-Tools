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
	"time"
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

// GetSystemHostname obtém e força estritamente o hostname do computador real.
// Impede qualquer alteração manual pelo usuário, arquivo de configuração ou variáveis de ambiente.
func GetSystemHostname() string {
	if h, err := os.Hostname(); err == nil && strings.TrimSpace(h) != "" {
		return strings.TrimSpace(h)
	}
	if h := os.Getenv("COMPUTERNAME"); strings.TrimSpace(h) != "" {
		return strings.TrimSpace(h)
	}
	if h := os.Getenv("HOSTNAME"); strings.TrimSpace(h) != "" {
		return strings.TrimSpace(h)
	}
	return "pc-windows"
}

// LoadConfig carrega as configurações com a seguinte prioridade:
// 1. Variáveis de ambiente (ou arquivo .env)
// 2. Arquivo config.json (se existir)
// 3. Valores padrão
// NOTA: O NodeID é SEMPRE forçado para o hostname real do computador.
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
		SyncWithCentral: true,
	}

	// 1. Tenta carregar do config.json se ele existir
	var rawData map[string]interface{}
	if data, err := os.ReadFile(targetPath); err == nil {
		_ = json.Unmarshal(data, cfg)
		_ = json.Unmarshal(data, &rawData)
	}

	// 2. Sobrescreve com variáveis do .env / Ambiente (têm prioridade)
	// PROBE_NODE_ID é ignorado intencionalmente para garantir integridade do hostname
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
	if v := os.Getenv("PROBE_SYNC_WITH_CENTRAL"); v != "" {
		cfg.SyncWithCentral = (v == "true" || v == "1" || v == "yes")
	} else if cfg.ServerURL != "" && cfg.ServerURL != "none" && rawData != nil {
		if val, exists := rawData["sync_with_central"]; exists {
			if b, ok := val.(bool); ok {
				cfg.SyncWithCentral = b
			}
		} else {
			cfg.SyncWithCentral = true
		}
	} else if cfg.ServerURL != "" && cfg.ServerURL != "none" {
		cfg.SyncWithCentral = true
	}

	// 3. Força ESTRITAMENTE o Hostname real do SO como NodeID (não pode mais ser alterado)
	cfg.NodeID = GetSystemHostname()

	if cfg.Location == "" {
		cfg.Location = "Escritorio-Local"
	}
	if cfg.ServerURL == "" {
		cfg.ServerURL = "https://nettools.acon.uepg.br"
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
	log.Printf("[CONFIG] Configurações ativas: NodeID=%s (Hostname Fixo), Location=%s, Server=%s, Interval=%dm, SyncCentral=%v, Ping=%s",
		cfg.NodeID, cfg.Location, cfg.ServerURL, cfg.IntervalMinutes, cfg.SyncWithCentral, cfg.PingTarget)

	return appConfig, nil
}

// GetCurrentConfig retorna a configuração atualmente carregada em memória
func GetCurrentConfig() Config {
	configMu.RLock()
	defer configMu.RUnlock()
	hostname := GetSystemHostname()
	if appConfig == nil {
		return Config{
			NodeID:          hostname,
			Location:        "Escritorio-Local",
			ServerURL:       "https://nettools.acon.uepg.br",
			APIToken:        "super-secret-probe-token-2026",
			IntervalMinutes: 60,
			PingTarget:      "1.1.1.1",
			SyncWithCentral: true,
		}
	}
	cfg := *appConfig
	cfg.NodeID = hostname
	return cfg
}

// SaveConfig atualiza o arquivo config.json e o arquivo .env
func SaveConfig(cfg Config) error {
	configMu.Lock()
	defer configMu.Unlock()

	// Preserva estritamente o hostname real da máquina (NodeID não pode ser alterado por ninguém)
	cfg.NodeID = GetSystemHostname()

	// 1. Atualiza config.json
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	executableDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	targetJsonPath := filepath.Join(executableDir, defaultConfigFilename)
	_ = os.WriteFile(defaultConfigFilename, data, 0644)
	if targetJsonPath != defaultConfigFilename {
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
		"PROBE_SYNC_WITH_CENTRAL=" + strconv.FormatBool(cfg.SyncWithCentral),
		"PROBE_PING_TARGET=" + cfg.PingTarget,
		"",
	}, "\n")

	targetEnvPath := filepath.Join(executableDir, defaultEnvFilename)
	_ = os.WriteFile(defaultEnvFilename, []byte(envContent), 0644)
	if targetEnvPath != defaultEnvFilename {
		_ = os.WriteFile(targetEnvPath, []byte(envContent), 0644)
	}

	appConfig = &cfg
	log.Printf("[CONFIG] Configurações e .env atualizados com sucesso")
	return nil
}

// getNextSyncSlot calcula o próximo horário fixo (:00 ou :30) do relógio de parede
func getNextSyncSlot(now time.Time) time.Time {
	var target time.Time
	if now.Minute() < 30 {
		target = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 30, 0, 0, now.Location())
	} else {
		nextHour := now.Add(1 * time.Hour)
		target = time.Date(nextHour.Year(), nextHour.Month(), nextHour.Day(), nextHour.Hour(), 0, 0, 0, now.Location())
	}
	if !target.After(now) {
		target = target.Add(30 * time.Minute)
	}
	return target
}
