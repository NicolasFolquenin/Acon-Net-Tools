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

	// 2. Informações de Hardware no Windows via WMI / PowerShell ou Linux via /proc
	if runtime.GOOS == "windows" {
		collectWindowsHardware(&info)
	} else if runtime.GOOS == "linux" {
		collectLinuxHardware(&info)
	}

	return info
}

func collectNetworkDetails(info *SystemInfo) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return
	}

	// 1. Tenta capturar Gateway e Interface primária via comando do sistema
	primaryAdapterName := ""
	if runtime.GOOS == "windows" {
		cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command",
			`$r = (Get-NetRoute -DestinationPrefix "0.0.0.0/0" -ErrorAction SilentlyContinue | Sort-Object RouteMetric | Select-Object -First 1); if ($r) { "$($r.NextHop)|$($r.InterfaceAlias)" }`)
		hideWindow(cmd)
		if out, err := cmd.Output(); err == nil {
			parts := strings.Split(strings.TrimSpace(string(out)), "|")
			if len(parts) >= 1 && parts[0] != "" {
				info.Gateway = parts[0]
			}
			if len(parts) >= 2 && parts[1] != "" {
				primaryAdapterName = strings.TrimSpace(parts[1])
			}
		}
	} else if runtime.GOOS == "linux" {
		collectLinuxGateway(info)
		primaryAdapterName = info.NetworkAdapter
	}

	var details []NetworkInterfaceDetail
	var primaryDetail *NetworkInterfaceDetail

	for _, iface := range interfaces {
		// Ignora interfaces de loopback
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		mac := strings.ToUpper(strings.ReplaceAll(iface.HardwareAddr.String(), ":", "-"))
		isUp := (iface.Flags & net.FlagUp) != 0

		var ipList []string
		if addrs, err := iface.Addrs(); err == nil {
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
					ipList = append(ipList, ip4.String())
				}
			}
		}

		// Se a placa não tem MAC, não tem IP e está down, ignoramos (adapters virtuais transientes)
		if mac == "" && len(ipList) == 0 && !isUp {
			continue
		}

		ifaceType := detectInterfaceType(iface.Name)
		isPrimary := false
		if primaryAdapterName != "" && (strings.EqualFold(iface.Name, primaryAdapterName) || strings.Contains(strings.ToLower(iface.Name), strings.ToLower(primaryAdapterName))) {
			isPrimary = true
		}

		detail := NetworkInterfaceDetail{
			Name:        iface.Name,
			Description: iface.Name,
			MACAddress:  mac,
			IPAddresses: ipList,
			Type:        ifaceType,
			IsPrimary:   isPrimary,
			IsUp:        isUp,
			Gateway:     info.Gateway,
		}

		details = append(details, detail)

		if isPrimary && primaryDetail == nil && len(ipList) > 0 {
			primaryDetail = &detail
		}
	}

	// Se nenhuma foi eleita como primária pela rota, seleciona a primeira interface UP que tenha IPv4 válido
	if primaryDetail == nil {
		for i := range details {
			if details[i].IsUp && len(details[i].IPAddresses) > 0 && details[i].MACAddress != "" {
				details[i].IsPrimary = true
				primaryDetail = &details[i]
				break
			}
		}
	}

	info.Interfaces = details

	// Preenche campos legados com a interface primária
	if primaryDetail != nil {
		if len(primaryDetail.IPAddresses) > 0 {
			info.IPAddress = primaryDetail.IPAddresses[0]
		}
		info.MACAddress = primaryDetail.MACAddress
		info.NetworkAdapter = primaryDetail.Name
	}
}

func detectInterfaceType(name string) string {
	lower := strings.ToLower(name)
	if strings.Contains(lower, "wi-fi") || strings.Contains(lower, "wifi") || strings.Contains(lower, "wireless") || strings.Contains(lower, "wlan") || strings.HasPrefix(lower, "wl") {
		return "wifi"
	}
	if strings.Contains(lower, "ethernet") || strings.Contains(lower, "eth") || strings.HasPrefix(lower, "en") || strings.HasPrefix(lower, "em") {
		return "ethernet"
	}
	if strings.Contains(lower, "veth") || strings.Contains(lower, "docker") || strings.Contains(lower, "br-") || strings.Contains(lower, "virtual") || strings.Contains(lower, "vpn") {
		return "virtual"
	}
	return "ethernet"
}

func collectLinuxGateway(info *SystemInfo) {
	// 1. Tenta via comando ip route show default
	cmd := exec.Command("ip", "route", "show", "default")
	if out, err := cmd.Output(); err == nil && len(out) > 0 {
		fields := strings.Fields(string(out))
		for i, field := range fields {
			if field == "via" && i+1 < len(fields) {
				info.Gateway = fields[i+1]
			}
			if field == "dev" && i+1 < len(fields) && info.NetworkAdapter == "" {
				info.NetworkAdapter = fields[i+1]
			}
		}
	}

	// 2. Fallback: leitura de /proc/net/route
	if info.Gateway == "" {
		if data, err := os.ReadFile("/proc/net/route"); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines[1:] {
				cols := strings.Fields(line)
				if len(cols) >= 3 && cols[1] == "00000000" { // Destino default 0.0.0.0
					if gwHex, err := parseHexIPv4(cols[2]); err == nil && gwHex != "0.0.0.0" {
						info.Gateway = gwHex
						if info.NetworkAdapter == "" {
							info.NetworkAdapter = cols[0]
						}
						break
					}
				}
			}
		}
	}
}

func parseHexIPv4(hexStr string) (string, error) {
	var ipBytes [4]byte
	n, err := fmt.Sscanf(hexStr, "%02x%02x%02x%02x", &ipBytes[3], &ipBytes[2], &ipBytes[1], &ipBytes[0])
	if err != nil || n != 4 {
		return "", fmt.Errorf("formato hex invalido")
	}
	return fmt.Sprintf("%d.%d.%d.%d", ipBytes[0], ipBytes[1], ipBytes[2], ipBytes[3]), nil
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

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", psScript)
	hideWindow(cmd)
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
	cmdAdapter := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command",
		`(Get-NetAdapter | Where-Object Status -eq 'Up' | Select-Object -First 1).InterfaceDescription`)
	hideWindow(cmdAdapter)
	if outA, errA := cmdAdapter.Output(); errA == nil && len(strings.TrimSpace(string(outA))) > 0 {
		info.NetworkAdapter = strings.TrimSpace(string(outA))
	}
}

func collectLinuxHardware(info *SystemInfo) {
	// 1. Nome do SO via /etc/os-release
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				val := strings.TrimPrefix(line, "PRETTY_NAME=")
				info.OS = strings.Trim(val, "\"")
				break
			}
		}
	}

	// 2. CPU via /proc/cpuinfo
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "model name") || strings.HasPrefix(line, "Hardware") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					info.CPU = strings.TrimSpace(parts[1])
					break
				}
			}
		}
	}

	// 3. RAM Total via /proc/meminfo
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					var kb float64
					if _, err := fmt.Sscanf(fields[1], "%f", &kb); err == nil && kb > 0 {
						gb := kb / (1024 * 1024)
						info.RAMTotalGB = float64(int(gb*10+0.5)) / 10.0
					}
				}
				break
			}
		}
	}

	// 4. Modelo de hardware DMI / Device-Tree (ex: Raspberry Pi)
	vendor := ""
	if vData, err := os.ReadFile("/sys/class/dmi/id/sys_vendor"); err == nil {
		vendor = strings.TrimSpace(string(vData))
	}
	product := ""
	if pData, err := os.ReadFile("/sys/class/dmi/id/product_name"); err == nil {
		product = strings.TrimSpace(string(pData))
	}
	if vendor != "" || product != "" {
		info.Model = strings.TrimSpace(vendor + " " + product)
	} else if dtModel, err := os.ReadFile("/proc/device-tree/model"); err == nil {
		info.Model = strings.TrimSpace(string(dtModel))
	}
}
