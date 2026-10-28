package handlers

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"central-server/db"
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

	ctx, cancel := context.WithTimeout(r.Context(), 14*time.Second)
	defer cancel()

	// Se o alvo não foi explicitado, localiza IP da máquina no banco
	if target == "" && hostname != "" {
		mach, err := h.db.GetMachine(ctx, hostname)
		if err == nil && mach != nil && mach.IPAddress != "" {
			target = mach.IPAddress
		} else {
			latest, errLatest := h.db.GetLatestMetricForNode(ctx, hostname)
			if errLatest == nil && latest != nil && latest.SystemInfo != nil && latest.SystemInfo.IPAddress != "" {
				target = latest.SystemInfo.IPAddress
			}
		}
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
	_ = json.NewEncoder(w).Encode(res)
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
		return
	}

	// Windows: Mínimo = 1ms, Máximo = 2ms, Média = 1ms
	reWinRtt := regexp.MustCompile(`(?i)(?:M[íi]nimo|Minimum)\s*=\s*([0-9.]+)\s*ms.*?(?:M[áa]ximo|Maximum)\s*=\s*([0-9.]+)\s*ms.*?(?:M[ée]dia|Average)\s*=\s*([0-9.]+)\s*ms`)
	if m := reWinRtt.FindStringSubmatch(output); len(m) >= 4 {
		resp.MinLatencyMs, _ = strconv.ParseFloat(m[1], 64)
		resp.MaxLatencyMs, _ = strconv.ParseFloat(m[2], 64)
		resp.AvgLatencyMs, _ = strconv.ParseFloat(m[3], 64)
		return
	}

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

	// Preenche aliases para total compatibilidade com o frontend
	resp.IsAlive = resp.Success
	resp.PacketLoss = resp.PacketLossPercent
	resp.MinMs = resp.MinLatencyMs
	resp.AvgMs = resp.AvgLatencyMs
	resp.MaxMs = resp.MaxLatencyMs
}

