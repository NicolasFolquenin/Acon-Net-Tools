package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ProfileManager gerencia a busca, cache local e ciclo de vida do perfil de medição individual por máquina
type ProfileManager struct {
	cachePath  string
	profile    *MeasurementProfile
	httpClient *http.Client
	mu         sync.RWMutex
}

// NewProfileManager instancia o gerenciador
func NewProfileManager() *ProfileManager {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "local-probe"
	}

	execDir, errExec := filepath.Abs(filepath.Dir(os.Args[0]))
	if errExec != nil {
		execDir = "."
	}
	cachePath := filepath.Join(execDir, fmt.Sprintf("profile_%s.json", hostname))

	pm := &ProfileManager{
		cachePath: cachePath,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	// Tenta carregar cache existente em disco para este hostname
	if cached, err := pm.loadFromDisk(); err == nil && cached != nil {
		pm.profile = cached
		log.Printf("[PROFILE] Perfil carregado do cache local (%s): versao %s (%d testes)", filepath.Base(cachePath), cached.Version, len(cached.Tests))
	} else {
		pm.profile = pm.GetBuiltinFallback()
		log.Printf("[PROFILE] Nenhum cache local para %s. Usando perfil embutido: versao %s", hostname, pm.profile.Version)
	}

	return pm
}

// FetchRemoteProfile tenta baixar o perfil mais recente para a máquina atual
func (pm *ProfileManager) FetchRemoteProfile(ctx context.Context, serverURL, apiToken string) (*MeasurementProfile, error) {
	hostname, _ := os.Hostname()
	return pm.FetchRemoteProfileForHost(ctx, serverURL, apiToken, HostBootstrapInfo{Hostname: hostname})
}

// FetchRemoteProfileForHost busca a configuração individual {hostname}.json no servidor central
// Fluxo:
// 1. GET /api/v1/probes/profile/{hostname}
// 2. Se HTTP 200: máquina já cadastrada com seu JSON -> aplica e salva no cache
// 3. Se HTTP 404: máquina nova -> realiza auto-registro POST /api/v1/probes/register
// 4. Se falha de rede/offline -> utiliza cache local em disco ou perfil fallback
func (pm *ProfileManager) FetchRemoteProfileForHost(ctx context.Context, serverURL, apiToken string, hostInfo HostBootstrapInfo) (*MeasurementProfile, error) {
	cleanURL := strings.TrimRight(strings.TrimSpace(serverURL), "/")
	if cleanURL == "" || cleanURL == "none" || cleanURL == "disabled" {
		return pm.GetActiveProfile(), nil
	}

	hostname := strings.TrimSpace(hostInfo.Hostname)
	if hostname == "" {
		hostname, _ = os.Hostname()
	}
	if hostname == "" {
		hostname = "local-probe"
	}

	// Atualiza caminho do cache para o host atual
	execDir, errExec := filepath.Abs(filepath.Dir(os.Args[0]))
	if errExec == nil {
		pm.mu.Lock()
		pm.cachePath = filepath.Join(execDir, fmt.Sprintf("profile_%s.json", hostname))
		pm.mu.Unlock()
	}

	// 1. Tenta buscar configuração individual por hostname
	url := fmt.Sprintf("%s/api/v1/probes/profile/%s", cleanURL, hostname)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return pm.GetActiveProfile(), err
	}

	if apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+apiToken)
	}

	resp, err := pm.httpClient.Do(req)
	if err != nil {
		log.Printf("[PROFILE][AVISO] Falha de conexao com %s (%v). Mantendo perfil em cache.", url, err)
		return pm.GetActiveProfile(), nil
	}
	defer resp.Body.Close()

	// 2. Se HTTP 200 OK -> Perfil individual já configurado no Central Server
	if resp.StatusCode == http.StatusOK {
		var prof MeasurementProfile
		if err := json.NewDecoder(resp.Body).Decode(&prof); err != nil {
			return pm.GetActiveProfile(), err
		}

		pm.saveToDisk(&prof)

		pm.mu.Lock()
		pm.profile = &prof
		pm.mu.Unlock()

		log.Printf("[PROFILE] Perfil individual sincronizado para a maquina '%s': versao %s (%d testes)",
			hostname, prof.Version, len(prof.Tests))
		return &prof, nil
	}

	// 3. Se HTTP 404 Not Found -> Máquina nova sem arquivo de configuração
	if resp.StatusCode == http.StatusNotFound {
		log.Printf("[PROFILE] Maquina '%s' identificada como nova pelo servidor (HTTP 404). Realizando auto-registro...", hostname)

		regPayload := ProbeRegisterRequest{
			Hostname:   hostname,
			IPAddress:  hostInfo.IPAddress,
			MACAddress: hostInfo.MACAddress,
			VLAN:       hostInfo.VLAN,
			SwitchName: hostInfo.SwitchName,
			SwitchIP:   hostInfo.SwitchIP,
			SwitchPort: hostInfo.SwitchPort,
		}

		regData, errJSON := json.Marshal(regPayload)
		if errJSON == nil {
			regURL := cleanURL + "/api/v1/probes/register"
			regReq, errRegReq := http.NewRequestWithContext(ctx, http.MethodPost, regURL, bytes.NewReader(regData))
			if errRegReq == nil {
				regReq.Header.Set("Content-Type", "application/json")
				if apiToken != "" {
					regReq.Header.Set("Authorization", "Bearer "+apiToken)
				}

				regResp, errReg := pm.httpClient.Do(regReq)
				if errReg == nil {
					defer regResp.Body.Close()
					if regResp.StatusCode == http.StatusCreated || regResp.StatusCode == http.StatusOK {
						var newProf MeasurementProfile
						if errDec := json.NewDecoder(regResp.Body).Decode(&newProf); errDec == nil {
							pm.saveToDisk(&newProf)

							pm.mu.Lock()
							pm.profile = &newProf
							pm.mu.Unlock()

							log.Printf("[PROFILE] Auto-registro concluido com sucesso! Perfil atribuido: versao %s (%d testes)",
								newProf.Version, len(newProf.Tests))
							return &newProf, nil
						}
					} else {
						log.Printf("[PROFILE][AVISO] Falha no auto-registro: servidor retornou status %d", regResp.StatusCode)
					}
				} else {
					log.Printf("[PROFILE][AVISO] Falha ao conectar para auto-registro: %v", errReg)
				}
			}
		}
	}

	return pm.GetActiveProfile(), nil
}

// GetActiveProfile retorna o perfil ativo em memória
func (pm *ProfileManager) GetActiveProfile() *MeasurementProfile {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.profile
}

func (pm *ProfileManager) loadFromDisk() (*MeasurementProfile, error) {
	pm.mu.RLock()
	path := pm.cachePath
	pm.mu.RUnlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var prof MeasurementProfile
	if err := json.Unmarshal(data, &prof); err != nil {
		return nil, err
	}
	return &prof, nil
}

func (pm *ProfileManager) saveToDisk(prof *MeasurementProfile) {
	data, err := json.MarshalIndent(prof, "", "  ")
	if err != nil {
		log.Printf("[PROFILE][AVISO] Falha ao serializar perfil para cache local: %v", err)
		return
	}
	pm.mu.RLock()
	path := pm.cachePath
	pm.mu.RUnlock()

	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Printf("[PROFILE][AVISO] Falha ao gravar cache do perfil em disco (%s): %v", path, err)
	}
}

// GetBuiltinFallback entrega perfil seguro se nem rede nem disco responderem
func (pm *ProfileManager) GetBuiltinFallback() *MeasurementProfile {
	return &MeasurementProfile{
		Version:         "2026.1-local-fallback",
		UpdatedAt:       time.Now().UTC(),
		IntervalMinutes: 30,
		SyncSchedule:    "fixed_half_hour",
		Description:     "Perfil padrão de fallback local da sonda UEPG",
		Tests: []TestTaskConfig{
			{
				ID:        "ping_gateway",
				Type:      TaskTypePing,
				Name:      "Latência Gateway Local",
				Enabled:   true,
				Target:    "auto_gateway",
				Count:     3,
				TimeoutMS: 1000,
			},
			{
				ID:        "ping_nti_core",
				Type:      TaskTypePing,
				Name:      "Latência Core NTI (Anel)",
				Enabled:   true,
				Target:    "177.101.22.40",
				Count:     3,
				TimeoutMS: 1000,
			},
			{
				ID:        "packet_loss_core",
				Type:      TaskTypePacketLoss,
				Name:      "Perda de Pacotes Core NTI",
				Enabled:   true,
				Target:    "177.101.22.40",
				Count:     10,
				TimeoutMS: 1000,
			},
			{
				ID:        "dns_uepg",
				Type:      TaskTypeDNSLookup,
				Name:      "Resolução DNS Institucional",
				Enabled:   true,
				Domain:    "uepg.br",
				TimeoutMS: 2000,
			},
			{
				ID:             "throughput_download",
				Type:           TaskTypeThroughputDownload,
				Name:           "Throughput Download",
				Enabled:        true,
				Endpoint:       "/api/v1/speedtest/download",
				SizeMB:         5,
				TimeoutSeconds: 15,
			},
			{
				ID:             "throughput_upload",
				Type:           TaskTypeThroughputUpload,
				Name:           "Throughput Upload",
				Enabled:        true,
				Endpoint:       "/api/v1/speedtest/upload",
				SizeMB:         2,
				TimeoutSeconds: 15,
			},
			{
				ID:                   "network_switch_discovery",
				Type:                 TaskTypeDiscoveryProtocol,
				Name:                 "Identificação Switch / Porta",
				Enabled:              true,
				Protocol:             "LLDP_CDP",
				ListenTimeoutSeconds: 45,
			},
			{
				ID:      "system_hardware_info",
				Type:    TaskTypeSystemInfo,
				Name:    "Telemetria de Host e Interface",
				Enabled: true,
			},
		},
	}
}
