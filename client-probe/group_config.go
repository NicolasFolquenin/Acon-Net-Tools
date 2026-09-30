package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	groupConfigMu sync.RWMutex
	cachedConfig  *ProbeConfigResponse
)

const cachedConfigFilename = "probe_group_config.json"

// FetchProbeGroupConfig consulta o Central Server para obter os módulos e diretrizes da VLAN atual.
// Em caso de falha de rede, recupera do cache local persistente em disco.
func FetchProbeGroupConfig(ctx context.Context, cfg Config, vlan int) (*ProbeConfigResponse, error) {
	groupConfigMu.Lock()
	defer groupConfigMu.Unlock()

	cleanURL := strings.TrimSpace(cfg.ServerURL)
	if cleanURL != "" && cleanURL != "none" && cleanURL != "disabled" {
		endpoint := fmt.Sprintf("%s/api/v1/config/probe?vlan=%d&hostname=%s",
			strings.TrimRight(cleanURL, "/"),
			vlan,
			url.QueryEscape(cfg.NodeID),
		)

		reqCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, endpoint, nil)
		if err == nil {
			if cfg.APIToken != "" {
				req.Header.Set("Authorization", "Bearer "+cfg.APIToken)
			}
			req.Header.Set("User-Agent", "UEPG-Net-Tools-Probe/2.0")

			client := &http.Client{Timeout: 6 * time.Second}
			resp, errDo := client.Do(req)
			if errDo == nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					var result ProbeConfigResponse
					if errDec := json.NewDecoder(resp.Body).Decode(&result); errDec == nil {
						log.Printf("[GROUP_CONFIG] Diretrizes recebidas com sucesso do servidor central: Grupo '%s' (VLAN %d, Intervalo: %dm)",
							result.GroupName, result.MatchedVLAN, result.IntervalMinutes)

						// Salva em cache local
						cachedConfig = &result
						saveGroupConfigCache(&result)
						return &result, nil
					}
				} else {
					log.Printf("[GROUP_CONFIG] Servidor central retornou HTTP %d na consulta de grupo", resp.StatusCode)
				}
			} else {
				log.Printf("[GROUP_CONFIG] Falha na requisição ao servidor central: %v. Usando cache local.", errDo)
			}
		}
	}

	// Fallback 1: Carrega cache do disco
	if cached := loadGroupConfigCache(); cached != nil {
		log.Printf("[GROUP_CONFIG] Usando configuração de grupo salva em cache: Grupo '%s'", cached.GroupName)
		cachedConfig = cached
		return cached, nil
	}

	// Fallback 2: Gera configuração padrão local
	log.Println("[GROUP_CONFIG] Sem conexão ou cache prévio. Aplicando módulos padrão de teste.")
	defaultResp := &ProbeConfigResponse{
		GroupID:         "local-fallback",
		GroupName:       "Grupo Local (Fallback)",
		MatchedVLAN:     vlan,
		IsDefault:       true,
		IntervalMinutes: cfg.IntervalMinutes,
		Modules:         DefaultModulesConfig(),
	}
	cachedConfig = defaultResp
	return defaultResp, nil
}

// GetCurrentGroupConfig retorna a última configuração de grupo obtida
func GetCurrentGroupConfig() *ProbeConfigResponse {
	groupConfigMu.RLock()
	defer groupConfigMu.RUnlock()

	if cachedConfig != nil {
		return cachedConfig
	}
	if disk := loadGroupConfigCache(); disk != nil {
		return disk
	}
	return &ProbeConfigResponse{
		GroupID:         "default",
		GroupName:       "Padrão",
		IsDefault:       true,
		IntervalMinutes: 60,
		Modules:         DefaultModulesConfig(),
	}
}

func saveGroupConfigCache(c *ProbeConfigResponse) {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(cachedConfigFilename, data, 0644)
	execDir, errExec := filepath.Abs(filepath.Dir(os.Args[0]))
	if errExec == nil && execDir != "." {
		_ = os.WriteFile(filepath.Join(execDir, cachedConfigFilename), data, 0644)
	}
}

func loadGroupConfigCache() *ProbeConfigResponse {
	var data []byte
	var err error

	execDir, errExec := filepath.Abs(filepath.Dir(os.Args[0]))
	if errExec == nil {
		data, err = os.ReadFile(filepath.Join(execDir, cachedConfigFilename))
	}
	if err != nil || len(data) == 0 {
		data, err = os.ReadFile(cachedConfigFilename)
	}
	if err != nil || len(data) == 0 {
		return nil
	}

	var res ProbeConfigResponse
	if err := json.Unmarshal(data, &res); err != nil {
		return nil
	}
	return &res
}
