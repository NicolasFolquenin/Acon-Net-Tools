package main

import (
	"context"
	"encoding/json"
	"log"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// DiscoverSwitchLLDP executa a captura de pacotes LLDP no Windows (PowerShell/PktMon) ou Linux (lldpcli).
// Se não houver broadcast LLDP na porta ou faltar privilégio de captura, retorna fallback seguro.
func DiscoverSwitchLLDP(ctx context.Context) SwitchInfo {
	if runtime.GOOS == "windows" {
		return discoverWindowsLLDP(ctx)
	} else if runtime.GOOS == "linux" {
		return discoverLinuxLLDP(ctx)
	}

	return SwitchInfo{
		Protocol:     "LLDP",
		SwitchName:   "Aguardando frame LLDP",
		SwitchPort:   "--",
		DiscoveredAt: time.Now().UTC(),
	}
}

func discoverWindowsLLDP(ctx context.Context) SwitchInfo {
	info := SwitchInfo{
		Protocol: "LLDP",
	}

	psScript := `
	$ErrorActionPreference = 'SilentlyContinue'

	# 1. Garante que o módulo PSDiscoveryProtocol esteja disponível (instala automaticamente se ausente)
	if (-not (Get-Module -ListAvailable -Name PSDiscoveryProtocol)) {
		try {
			[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
			Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass -Force -ErrorAction SilentlyContinue
			Install-PackageProvider -Name NuGet -MinimumVersion 2.8.5.201 -Force -Scope CurrentUser -ErrorAction SilentlyContinue | Out-Null
			Install-Module -Name PSDiscoveryProtocol -Scope CurrentUser -Force -SkipPublisherCheck -AllowClobber -ErrorAction SilentlyContinue | Out-Null
		} catch {}
	}

	# 2. Tenta capturar dados via PSDiscoveryProtocol (LLDP e CDP de switches Cisco/HP/Aruba/Huawei/Mikrotik)
	if (Get-Module -ListAvailable -Name PSDiscoveryProtocol) {
		try {
			Import-Module PSDiscoveryProtocol -ErrorAction SilentlyContinue
			$capture = Invoke-DiscoveryProtocolCapture -Duration 35 -Force -ErrorAction SilentlyContinue
			if ($capture) {
				$rawList = $capture | Get-DiscoveryProtocolData -ErrorAction SilentlyContinue
				if ($rawList) {
					# Filtra ESTRITAMENTE apenas SWITCHES de rede reais, descartando telefones VoIP e dispositivos SEP
					$realSwitches = @($rawList) | Where-Object {
						$dev = if ($_.Device) { [string]$_.Device } elseif ($_.SysName) { [string]$_.SysName } else { "" }
						$model = if ($_.Model) { [string]$_.Model } elseif ($_.Type) { [string]$_.Type } else { "" }
						$dev -notmatch '^SEP[0-9A-Fa-f]{12}' -and $model -notmatch 'Phone|Telefone|VoIP'
					}

					if ($realSwitches) {
						$sorted = $realSwitches | Sort-Object -Descending -Property {
							$score = 0
							$port = if ($_.Port) { [string]$_.Port } else { "" }
							$vlan = if ($_.VLAN) { [int]$_.VLAN } else { 0 }
							if ($vlan -gt 0) { $score += 50 }
							if ($port -match '^(GE|Gi|Fa|Te|Eth|TenGig|Gigabit|Port-channel|swp)') { $score += 30 }
							$score
						}

						$data = $sorted[0]
						$swName = if ($data.Device) { [string]$data.Device } elseif ($data.SysName) { [string]$data.SysName } else { "" }
						$swPort = if ($data.Port) { [string]$data.Port } else { "" }
						$swModel = if ($data.Model) { [string]$data.Model } elseif ($data.Description) { [string]$data.Description } elseif ($data.SysDescr) { [string]$data.SysDescr } elseif ($data.Type) { [string]$data.Type } else { "" }
						$swVlan = if ($data.VLAN) { [int]$data.VLAN } else { 0 }

						if ($swModel -match 'S5735-[A-Za-z0-9\-]+') {
							$swModel = "Huawei CloudEngine " + $Matches[0]
						} elseif ($swModel -match 'Huawei|CloudEngine' -and $swModel -notmatch 'CloudEngine') {
							$swModel = "Huawei " + $swModel
						}

						if ($swName -or $swPort) {
							[PSCustomObject]@{
								SwitchName = if ($swName) { $swName } else { "Switch Corporativo" }
								Port       = if ($swPort) { $swPort } else { "--" }
								Vlan       = $swVlan
								Model      = if ($swModel) { $swModel } else { "Huawei CloudEngine S5735" }
							} | ConvertTo-Json -Compress
							exit 0
						}
					}
				}
			}
		} catch {}
	}

	# 2. Tenta WMI nativo MSFT_NetLldpAgent se ativo
	try {
		$lldp = Get-CimInstance -Namespace root\StandardCimv2 -ClassName MSFT_NetLldpAgent -ErrorAction SilentlyContinue
		if ($lldp) {
			[PSCustomObject]@{
				SwitchName = $lldp.ChassisId
				Port       = $lldp.PortId
				Vlan       = $lldp.VlanId
				Model      = "Switch Gerenciável"
			} | ConvertTo-Json -Compress
			exit 0
		}
	} catch {}

	# Retorno padrao vazio caso nao haja anuncio no momento
	"{}"
	`

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", psScript)
	hideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		log.Printf("[LLDP] Consulta LLDP finalizada sem pacotes capturados: %v", err)
		info.SwitchName = "Não detectado (Sem broadcast LLDP)"
		info.SwitchPort = "--"
		info.DiscoveredAt = time.Now().UTC()
		return info
	}

	outStr := strings.TrimSpace(string(out))
	if outStr == "" || outStr == "{}" {
		info.SwitchName = "Aguardando frame LLDP"
		info.SwitchPort = "--"
		info.DiscoveredAt = time.Now().UTC()
		return info
	}

	var parsed struct {
		SwitchName string `json:"SwitchName"`
		Port       string `json:"Port"`
		Vlan       int    `json:"Vlan"`
		Model      string `json:"Model"`
	}

	if errJson := json.Unmarshal([]byte(outStr), &parsed); errJson == nil {
		if parsed.SwitchName != "" {
			info.SwitchName = parsed.SwitchName
		}
		if parsed.Port != "" {
			info.SwitchPort = parsed.Port
		}
		if parsed.Vlan > 0 {
			info.VLAN = parsed.Vlan
		}
		if parsed.Model != "" {
			info.SwitchModel = parsed.Model
		}
	}

	if info.SwitchName == "" {
		info.SwitchName = "Aguardando frame LLDP"
		info.SwitchPort = "--"
	}

	info.DiscoveredAt = time.Now().UTC()
	return info
}

func discoverLinuxLLDP(ctx context.Context) SwitchInfo {
	info := SwitchInfo{
		Protocol:     "LLDP",
		SwitchName:   "Aguardando frame LLDP",
		SwitchPort:   "--",
		DiscoveredAt: time.Now().UTC(),
	}

	// 1. Verifica se lldpcli está disponível no sistema (pacote lldpd)
	lldpPath, err := exec.LookPath("lldpcli")
	if err != nil {
		log.Printf("[LLDP] lldpcli não instalado no Linux (apt install lldpd para ativar captura LLDP)")
		return info
	}

	cmd := exec.CommandContext(ctx, lldpPath, "show", "neighbors", "-f", "json")
	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		return info
	}

	// Parser genérico para JSON do lldpcli
	// Estrutura padrão: { "lldp": { "interface": [ { "name": "eth0", "chassis": { ... }, "port": { ... }, "vlan": { ... } } ] } }
	var root map[string]interface{}
	if err := json.Unmarshal(out, &root); err != nil {
		return info
	}

	lldpObj, ok := root["lldp"].(map[string]interface{})
	if !ok {
		return info
	}

	// lldp.interface pode ser um mapa de interfaces ou slice
	var interfaces []map[string]interface{}
	if ifaces, ok := lldpObj["interface"].([]interface{}); ok {
		for _, item := range ifaces {
			if m, ok := item.(map[string]interface{}); ok {
				interfaces = append(interfaces, m)
			}
		}
	} else if ifaceMap, ok := lldpObj["interface"].(map[string]interface{}); ok {
		for _, val := range ifaceMap {
			if m, ok := val.(map[string]interface{}); ok {
				interfaces = append(interfaces, m)
			}
		}
	}

	for _, iface := range interfaces {
		chassis, _ := iface["chassis"].(map[string]interface{})
		port, _ := iface["port"].(map[string]interface{})
		vlan, _ := iface["vlan"].(map[string]interface{})

		swName := ""
		if chassis != nil {
			if nameObj, ok := chassis["name"].(map[string]interface{}); ok {
				swName, _ = nameObj["value"].(string)
			} else if descObj, ok := chassis["descr"].(map[string]interface{}); ok {
				info.SwitchModel, _ = descObj["value"].(string)
			}
		}

		swPort := ""
		if port != nil {
			if idObj, ok := port["id"].(map[string]interface{}); ok {
				swPort, _ = idObj["value"].(string)
			} else if descrObj, ok := port["descr"].(map[string]interface{}); ok {
				swPort, _ = descrObj["value"].(string)
			}
		}

		if vlan != nil {
			if vlanId, ok := vlan["vlan-id"].(float64); ok {
				info.VLAN = int(vlanId)
			}
		}

		if swName != "" || swPort != "" {
			if swName != "" {
				info.SwitchName = swName
			}
			if swPort != "" {
				info.SwitchPort = swPort
			}
			break
		}
	}

	return info
}
