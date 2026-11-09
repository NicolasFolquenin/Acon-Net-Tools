package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"central-server/db"
	"central-server/models"
)

// PingHandler gerencia requisições administrativas para teste de conectividade ICMP
type PingHandler struct {
	db *db.MongoDBClient
}

// NewPingHandler inicializa uma nova instância de PingHandler
func NewPingHandler(database *db.MongoDBClient) *PingHandler {
	return &PingHandler{db: database}
}

// PingRequest payload recebido para disparo do teste de ping
type PingRequest struct {
	Hostname string `json:"hostname"`
	Target   string `json:"target"`
}

// PingResponse resposta estruturada com métricas de conectividade e output bruto
type PingResponse struct {
	Target            string    `json:"target"`
	Hostname          string    `json:"hostname"`
	Success           bool      `json:"success"`
	IsAlive           bool      `json:"is_alive"`
	PacketsSent       int       `json:"packets_sent"`
	PacketsReceived   int       `json:"packets_received"`
	PacketLossPercent float64   `json:"packet_loss_percent"`
	PacketLoss        float64   `json:"packet_loss"`
	MinLatencyMs      float64   `json:"min_latency_ms"`
	AvgLatencyMs      float64   `json:"avg_latency_ms"`
	MaxLatencyMs      float64   `json:"max_latency_ms"`
	MinMs             float64   `json:"min_ms"`
	AvgMs             float64   `json:"avg_ms"`
	MaxMs             float64   `json:"max_ms"`
	RawOutput         string    `json:"raw_output"`
	Timestamp         time.Time `json:"timestamp"`
}

// AdminPing lida com POST/GET /api/v1/admin/ping
func (h *PingHandler) AdminPing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, `{"error": "metodo nao permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req PingRequest
	if r.Method == http.MethodPost {
		_ = json.NewDecoder(r.Body).Decode(&req)
	} else {
		req.Hostname = r.URL.Query().Get("hostname")
		req.Target = r.URL.Query().Get("target")
	}

	hostname := strings.TrimSpace(req.Hostname)
	target := strings.TrimSpace(req.Target)

	ctx, cancel := context.WithTimeout(r.Context(), 18*time.Second)
	defer cancel()

	// Se o alvo não foi explicitado, localiza IP ativo e válido da máquina no banco
	if target == "" && hostname != "" {
		target = h.resolveBestMachineIP(ctx, hostname)
		if target == "" {
			target = hostname
		}
	}

	if target == "" {
		http.Error(w, `{"error": "alvo ou hostname de destino obrigatorio"}`, http.StatusBadRequest)
		return
	}

	// Sanitização rigorosa para evitar command injection
	if !isValidPingTarget(target) {
		http.Error(w, `{"error": "formato de endereco de destino invalido"}`, http.StatusBadRequest)
		return
	}

	res := executePing(ctx, target, hostname)

	// Se a máquina não respondeu (100% de perda) e possui outros IPs ativos (ex: Wi-Fi), testa o IP alternativo
	if !res.Success && hostname != "" {
		altIPs := h.findAlternativeIPs(ctx, hostname, target)
		for _, alt := range altIPs {
			resAlt := executePing(ctx, alt, hostname)
			if resAlt.Success {
				resAlt.RawOutput = fmt.Sprintf("[IP Principal %s sem resposta; alternado automaticamente para interface ativa %s]\n\n", target, alt) + resAlt.RawOutput
				res = resAlt
				target = alt
				break
			}
		}
	}

	// Se ainda assim deu 100% de perda, anexa instrução de diagnóstico de firewall do Windows
	if !res.Success {
		res.RawOutput += "\n----------------------------------------------------------------------\n" +
			"[Diagnóstico de Conectividade ICMP]\n" +
			"• Host não respondeu aos pacotes de ping.\n" +
			"• No Windows 10/11, o Windows Defender Firewall bloqueia ping (ICMP) por padrão.\n" +
			"• Para liberar respostas de ping nesta máquina, execute no PowerShell (Admin):\n" +
			"  Enable-NetFirewallRule -Name 'FPS-ICMP4-ERQ-In'\n" +
			"• Se a máquina mudou para o Wi-Fi, confirme se a sonda executou um teste para atualizar o IP.\n"
	}

	// Registra a medição no banco de dados MongoDB para alimentar gráficos e tabela de telemetria
	if hostname != "" {
		h.recordPingMetric(ctx, hostname, target, res)
	}

	_ = json.NewEncoder(w).Encode(res)
}

func (h *PingHandler) resolveBestMachineIP(ctx context.Context, hostname string) string {
	if h.db == nil || hostname == "" {
		return ""
	}

	// 1. Tenta última medição (tem a foto mais recente das placas e do Wi-Fi)
	if latest, errLatest := h.db.GetLatestMetricForNode(ctx, hostname); errLatest == nil && latest != nil && latest.SystemInfo != nil {
		sys := latest.SystemInfo
		// Se estiver no Wi-Fi ou possuir placas ativas, busca interface operacional com IP
		for _, iface := range sys.Interfaces {
			if iface.IsUp && (iface.IsPrimary || iface.Type == "wifi" || strings.Contains(strings.ToLower(iface.Name), "wi-fi")) {
				for _, ip := range iface.IPAddresses {
					if isValidPingTarget(ip) {
						return ip
					}
				}
			}
		}
		if sys.IPAddress != "" && isValidPingTarget(sys.IPAddress) {
			return sys.IPAddress
		}
	}

	// 2. Tenta cadastro da máquina
	if mach, err := h.db.GetMachine(ctx, hostname); err == nil && mach != nil {
		if mach.SystemInfo != nil {
			for _, iface := range mach.SystemInfo.Interfaces {
				if iface.IsUp && iface.IsPrimary {
					for _, ip := range iface.IPAddresses {
						if isValidPingTarget(ip) {
							return ip
						}
					}
				}
			}
		}
		if mach.IPAddress != "" && isValidPingTarget(mach.IPAddress) {
			return mach.IPAddress
		}
	}

	return ""
}

func (h *PingHandler) findAlternativeIPs(ctx context.Context, hostname, currentTarget string) []string {
	if h.db == nil || hostname == "" {
		return nil
	}

	var ips []string
	seen := make(map[string]bool)
	seen[currentTarget] = true

	collectFromSys := func(sys *models.SystemInfo) {
		if sys == nil {
			return
		}
		if sys.IPAddress != "" && !seen[sys.IPAddress] && isValidPingTarget(sys.IPAddress) {
			ips = append(ips, sys.IPAddress)
			seen[sys.IPAddress] = true
		}
		for _, iface := range sys.Interfaces {
			if iface.IsUp {
				for _, ip := range iface.IPAddresses {
					if !seen[ip] && isValidPingTarget(ip) {
						ips = append(ips, ip)
						seen[ip] = true
					}
				}
			}
		}
	}

	if latest, errLatest := h.db.GetLatestMetricForNode(ctx, hostname); errLatest == nil && latest != nil {
		collectFromSys(latest.SystemInfo)
	}
	if mach, err := h.db.GetMachine(ctx, hostname); err == nil && mach != nil {
		if mach.IPAddress != "" && !seen[mach.IPAddress] && isValidPingTarget(mach.IPAddress) {
			ips = append(ips, mach.IPAddress)
			seen[mach.IPAddress] = true
		}
		collectFromSys(mach.SystemInfo)
	}

	return ips
}

func (h *PingHandler) recordPingMetric(ctx context.Context, hostname, target string, res PingResponse) {
	if h.db == nil || hostname == "" {
		return
	}

	location := "Geral"
	groupName := "default"
	vlan := 0
	clientVer := ""
	loggedUser := ""
	var sysInfo *models.SystemInfo
	var swInfo *models.SwitchInfo

	if mach, err := h.db.GetMachine(ctx, hostname); err == nil && mach != nil {
		if mach.GroupID != "" {
			groupName = mach.GroupID
		}
		if mach.VLAN > 0 {
			vlan = mach.VLAN
		}
		if mach.SystemInfo != nil {
			sysInfo = mach.SystemInfo
		}
		if mach.LoggedUser != "" {
			loggedUser = mach.LoggedUser
		}
	}

	if latest, errLatest := h.db.GetLatestMetricForNode(ctx, hostname); errLatest == nil && latest != nil {
		if location == "Geral" && latest.Location != "" {
			location = latest.Location
		}
		if groupName == "default" && latest.GroupName != "" {
			groupName = latest.GroupName
		}
		if vlan == 0 && latest.VLAN > 0 {
			vlan = latest.VLAN
		}
		if clientVer == "" && latest.ClientVersion != "" {
			clientVer = latest.ClientVersion
		}
		if loggedUser == "" && latest.LoggedUser != "" {
			loggedUser = latest.LoggedUser
		}
		if sysInfo == nil && latest.SystemInfo != nil {
			sysInfo = latest.SystemInfo
		}
		if swInfo == nil && latest.SwitchInfo != nil {
			swInfo = latest.SwitchInfo
		}
	}

	statusStr := "ok"
	if !res.Success || res.PacketLossPercent >= 100 {
		statusStr = "error"
	} else if res.PacketLossPercent > 0 {
		statusStr = "warning"
	}

	pingVal := res.AvgLatencyMs
	if !res.Success {
		pingVal = 0
	}

	metricsMap := map[string]interface{}{
		"ping_gateway": map[string]interface{}{
			"value":   pingVal,
			"unit":    "ms",
			"status":  statusStr,
			"details": fmt.Sprintf("Ping ICMP administrativo: %.1fms (%s)", pingVal, target),
		},
		"packet_loss_gateway": map[string]interface{}{
			"value":   res.PacketLossPercent,
			"unit":    "%",
			"status":  statusStr,
			"details": fmt.Sprintf("Perda ICMP administrativo: %.0f%% (%d/%d)", res.PacketLossPercent, res.PacketsReceived, res.PacketsSent),
		},
	}

	modResults := &models.ModularResults{
		PacketLoss: &models.PacketLossResult{
			Target:       target,
			Sent:         res.PacketsSent,
			Received:     res.PacketsReceived,
			LossPercent:  res.PacketLossPercent,
			MinLatencyMs: res.MinLatencyMs,
			AvgLatencyMs: res.AvgLatencyMs,
			MaxLatencyMs: res.MaxLatencyMs,
		},
	}

	metricInput := &models.MetricPayloadInput{
		NodeID:         hostname,
		Location:       location,
		GroupName:      groupName,
		VLAN:           vlan,
		Timestamp:      res.Timestamp.UTC(),
		PingGateway:    pingVal,
		ClientVersion:  clientVer,
		LoggedUser:     loggedUser,
		Metrics:        metricsMap,
		SystemInfo:     sysInfo,
		SwitchInfo:     swInfo,
		ModularResults: modResults,
	}

	if err := h.db.InsertMetric(ctx, metricInput); err != nil {
		log.Printf("[PING][ERRO] Falha ao persistir metrica de ping administrativo para %s: %v", hostname, err)
	} else {
		log.Printf("[PING][SUCESSO] Metrica de ping registrada no banco para %s: Latencia=%.1fms, Perda=%.0f%%", hostname, pingVal, res.PacketLossPercent)
	}
}

// isValidPingTarget valida se o alvo é um IPv4, IPv6 ou hostname RFC válido
func isValidPingTarget(target string) bool {
	if net.ParseIP(target) != nil {
		return true
	}
	re := regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)*[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?$`)
	return re.MatchString(target)
}

// executePing dispara comando ping nativo de acordo com o sistema operacional
func executePing(ctx context.Context, target, hostname string) PingResponse {
	resp := PingResponse{
		Target:      target,
		Hostname:    hostname,
		PacketsSent: 4,
		Timestamp:   time.Now().UTC(),
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "ping", "-n", "4", "-w", "2000", target)
	} else {
		cmd = exec.CommandContext(ctx, "ping", "-c", "4", "-W", "2", target)
	}

	outBytes, _ := cmd.CombinedOutput()
	outputStr := string(outBytes)
	resp.RawOutput = outputStr

	parsePingOutput(outputStr, &resp)
	return resp
}

// parsePingOutput analisa a saída textual do comando ping e preenche os campos estruturados
func parsePingOutput(output string, resp *PingResponse) {
	// 1. Perda de pacotes e contagem
	// Linux: 4 packets transmitted, 4 received, 0% packet loss
	reLinuxLoss := regexp.MustCompile(`(\d+)\s+packets transmitted,\s*(\d+)\s+received.*?(?:([0-9.]+)\s*%\s*packet loss)`)
	if m := reLinuxLoss.FindStringSubmatch(output); len(m) >= 4 {
		resp.PacketsSent, _ = strconv.Atoi(m[1])
		resp.PacketsReceived, _ = strconv.Atoi(m[2])
		resp.PacketLossPercent, _ = strconv.ParseFloat(m[3], 64)
	} else {
		// Windows: Enviados = 4, Recebidos = 4, Perdidos = 0 (0% de perda)
		// Ou em inglês: Sent = 4, Received = 4, Lost = 0 (0% loss)
		reWin := regexp.MustCompile(`(?i)(?:Enviados|Sent)\s*=\s*(\d+).*?(?:Recebidos|Received)\s*=\s*(\d+).*?(?:(?:Perdidos|Lost)\s*=\s*\d+\s*)?\(\s*([0-9.]+)\s*%\s*(?:de perda|loss)`)
		if m := reWin.FindStringSubmatch(output); len(m) >= 4 {
			resp.PacketsSent, _ = strconv.Atoi(m[1])
			resp.PacketsReceived, _ = strconv.Atoi(m[2])
			resp.PacketLossPercent, _ = strconv.ParseFloat(m[3], 64)
		}
	}

	if resp.PacketsReceived > 0 {
		resp.Success = true
	}

	// 2. Latência
	// Linux: rtt min/avg/max/mdev = 0.412/0.446/0.489/0.029 ms
	reLinuxRtt := regexp.MustCompile(`(?:rtt|round-trip)\s+(?:min/avg/max|min/avg/max/mdev|min/avg/max/stddev)\s*=\s*([0-9.]+)/([0-9.]+)/([0-9.]+)`)
	if m := reLinuxRtt.FindStringSubmatch(output); len(m) >= 4 {
		resp.MinLatencyMs, _ = strconv.ParseFloat(m[1], 64)
		resp.AvgLatencyMs, _ = strconv.ParseFloat(m[2], 64)
		resp.MaxLatencyMs, _ = strconv.ParseFloat(m[3], 64)
	} else {
		// Windows: Mínimo = 1ms, Máximo = 2ms, Média = 1ms
		reWinRtt := regexp.MustCompile(`(?i)(?:M[íi]nimo|Minimum)\s*=\s*([0-9.]+)\s*ms.*?(?:M[áa]ximo|Maximum)\s*=\s*([0-9.]+)\s*ms.*?(?:M[ée]dia|Average)\s*=\s*([0-9.]+)\s*ms`)
		if m := reWinRtt.FindStringSubmatch(output); len(m) >= 4 {
			resp.MinLatencyMs, _ = strconv.ParseFloat(m[1], 64)
			resp.MaxLatencyMs, _ = strconv.ParseFloat(m[2], 64)
			resp.AvgLatencyMs, _ = strconv.ParseFloat(m[3], 64)
		} else {
			// Fallback para linhas individuais: time=X.X ms ou tempo=Xms ou tempo<1ms
			reSingle := regexp.MustCompile(`(?i)(?:time|tempo)[=<]([0-9.]+)\s*ms`)
			matches := reSingle.FindAllStringSubmatch(output, -1)
			if len(matches) > 0 {
				var minLat, maxLat, sumLat float64
				minLat = 999999
				validCount := 0
				for _, m := range matches {
					if val, err := strconv.ParseFloat(m[1], 64); err == nil {
						if val < minLat {
							minLat = val
						}
						if val > maxLat {
							maxLat = val
						}
						sumLat += val
						validCount++
					}
				}
				if validCount > 0 {
					resp.MinLatencyMs = minLat
					resp.MaxLatencyMs = maxLat
					resp.AvgLatencyMs = sumLat / float64(validCount)
				}
			}
		}
	}

	// Preenche aliases para total compatibilidade com o frontend
	resp.IsAlive = resp.Success
	resp.PacketLoss = resp.PacketLossPercent
	resp.MinMs = resp.MinLatencyMs
	resp.AvgMs = resp.AvgLatencyMs
	resp.MaxMs = resp.MaxLatencyMs
}

