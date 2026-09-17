package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// CollectSystemInfo reúne informações detalhadas da máquina local de forma nativa e resiliente.
func CollectSystemInfo() SystemInfo {
	hostname, _ := os.Hostname()
	info := SystemInfo{
		Hostname: hostname,
		OS:       fmt.Sprintf("%s (%s)", runtime.GOOS, runtime.GOARCH),
	}

	// 1. Informações de Rede (IP Local, Gateway e MAC Address)
	collectNetworkDetails(&info)

	// 2. Informações de Hardware no Windows via WMI / PowerShell
	if runtime.GOOS == "windows" {
		collectWindowsHardware(&info)
	}

	return info
}

func collectNetworkDetails(info *SystemInfo) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return
	}

	for _, iface := range interfaces {
		// Ignora loopback e interfaces inativas
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		mac := iface.HardwareAddr.String()
		if mac == "" {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.IsLoopback() {
				continue
			}
			ip4 := ipNet.IP.To4()
			if ip4 != nil {
				// Evita endereços de autoconfiguração 169.254.x.x
				if ip4[0] == 169 && ip4[1] == 254 {
					continue
				}
				info.IPAddress = ip4.String()
				info.MACAddress = strings.ToUpper(strings.ReplaceAll(mac, ":", "-"))
				info.NetworkAdapter = iface.Name
				break
			}
		}

		if info.IPAddress != "" {
			break
		}
	}

	// Tenta capturar Gateway via comando do sistema
	if runtime.GOOS == "windows" {
		cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
			`(Get-NetRoute -DestinationPrefix "0.0.0.0/0" -ErrorAction SilentlyContinue | Select-Object -First 1).NextHop`)
		if out, err := cmd.Output(); err == nil {
			info.Gateway = strings.TrimSpace(string(out))
		}
	}
}

func collectWindowsHardware(info *SystemInfo) {
	psScript := `
	$ErrorActionPreference = 'SilentlyContinue'
	$cs = Get-CimInstance Win32_ComputerSystem
	$cpu = Get-CimInstance Win32_Processor | Select-Object -First 1
	$os = Get-CimInstance Win32_OperatingSystem

	[PSCustomObject]@{
		Model = "$($cs.Manufacturer) $($cs.Model)".Trim()
		CPU = $cpu.Name.Trim()
		RAM_GB = [math]::Round($cs.TotalPhysicalMemory / 1GB, 1)
		OSName = $os.Caption.Trim()
	} | ConvertTo-Json -Compress
	`

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
	out, err := cmd.Output()
	if err == nil && len(out) > 0 {
		var result struct {
			Model  string  `json:"Model"`
			CPU    string  `json:"CPU"`
			RAM_GB float64 `json:"RAM_GB"`
			OSName string  `json:"OSName"`
		}
		if errJson := json.Unmarshal(out, &result); errJson == nil {
			if result.Model != "" {
				info.Model = result.Model
			}
			if result.CPU != "" {
				info.CPU = result.CPU
			}
			if result.RAM_GB > 0 {
				info.RAMTotalGB = result.RAM_GB
			}
			if result.OSName != "" {
				info.OS = result.OSName
			}
		}
	}

	// Fallback para nome da interface descritiva
	cmdAdapter := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
		`(Get-NetAdapter | Where-Object Status -eq 'Up' | Select-Object -First 1).InterfaceDescription`)
	if outA, errA := cmdAdapter.Output(); errA == nil && len(strings.TrimSpace(string(outA))) > 0 {
		info.NetworkAdapter = strings.TrimSpace(string(outA))
	}
}
