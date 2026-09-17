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

	# 1. Tenta PSDiscoveryProtocol se instalado
	if (Get-Module -ListAvailable -Name PSDiscoveryProtocol) {
		try {
			$data = Invoke-DiscoveryProtocolCapture -Type LLDP -Duration 10 | Get-DiscoveryProtocolData
			if ($data) {
				[PSCustomObject]@{
					SwitchName = $data.Device
					Port = $data.Port
					Vlan = $data.Vlan
					Model = $data.Model
				} | ConvertTo-Json -Compress
				exit 0
			}
		} catch {}
	}

	# 2. Tenta WMI nativo MSFT_NetLldpAgent se ativo
	try {
		$lldp = Get-CimInstance -Namespace root\StandardCimv2 -ClassName MSFT_NetLldpAgent -ErrorAction SilentlyContinue
		if ($lldp) {
			[PSCustomObject]@{
				SwitchName = $lldp.ChassisId
				Port = $lldp.PortId
				Vlan = $lldp.VlanId
				Model = "Switch Gerenciável"
			} | ConvertTo-Json -Compress
			exit 0
		}
	} catch {}

	# Retorno padrão vazio caso não haja anúncio no momento
	"{}"
	`

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
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
