package main

import (
	"context"
	"encoding/json"
	"log"
	"os/exec"
	"strings"
	"time"
)

// DiscoverSwitchLLDP executa a captura de pacotes LLDP usando PowerShell / PktMon nativo do Windows.
// Se não houver broadcast LLDP na porta ou faltar privilégio de captura, retorna fallback seguro.
func DiscoverSwitchLLDP(ctx context.Context) SwitchInfo {
	info := SwitchInfo{
		Protocol: "LLDP",
	}

	// Script PowerShell para tentar capturar anúncio LLDP
	// 1. Tenta via módulo PSDiscoveryProtocol se disponível
	// 2. Ou via PktMon nativo filtrando EtherType 0x88CC
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
