package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// RunPingModule executa medição de latência ICMP/TCP para o alvo configurado
func RunPingModule(ctx context.Context, p *NetworkProbe, cfg PingModuleConfig) float64 {
	if !cfg.Enabled {
		return 0
	}
	target := strings.TrimSpace(cfg.Target)
	if target == "" {
		target = "1.1.1.1"
	}

	count := cfg.Count
	if count <= 0 {
		count = 4
	}

	timeoutMs := cfg.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = 1000
	}

	log.Printf("[MODULE:PING] Testando latência para %s (pacotes: %d)...", target, count)

	// 1. Tenta ping nativo via executável do SO
	if ms, err := p.pingViaExecWithParams(target, count, timeoutMs); err == nil && ms > 0 {
		return math.Round(ms*100) / 100
	}

	// 2. Fallback via TCP handshake
	ms := p.measurePingFallback(target)
	return math.Round(ms*100) / 100
}

// RunPacketLossModule envia N pacotes e calcula perda de pacotes e jitter (variação de latência)
func RunPacketLossModule(ctx context.Context, p *NetworkProbe, cfg PacketLossModuleConfig) *PacketLossResult {
	if !cfg.Enabled {
		return nil
	}
	target := strings.TrimSpace(cfg.Target)
	if target == "" {
		target = "1.1.1.1"
	}
	count := cfg.Count
	if count <= 0 {
		count = 10
	}
	timeoutMs := cfg.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = 1000
	}

	log.Printf("[MODULE:PACKET_LOSS] Medindo perda de pacotes e jitter para %s (amostras: %d)...", target, count)

	var latencies []float64
	received := 0

	for i := 0; i < count; i++ {
		select {
		case <-ctx.Done():
			break
		default:
		}

		start := time.Now()
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(target, "80"), time.Duration(timeoutMs)*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			lat := float64(time.Since(start).Microseconds()) / 1000.0
			latencies = append(latencies, lat)
			received++
		} else {
			// Tenta porta 53 (DNS)
			startDns := time.Now()
			connDns, errDns := net.DialTimeout("tcp", net.JoinHostPort(target, "53"), time.Duration(timeoutMs)*time.Millisecond)
			if errDns == nil {
				_ = connDns.Close()
				lat := float64(time.Since(startDns).Microseconds()) / 1000.0
				latencies = append(latencies, lat)
				received++
			}
		}

		time.Sleep(30 * time.Millisecond)
	}

	lossPercent := 0.0
	if count > 0 {
		lossPercent = float64(count-received) / float64(count) * 100.0
	}

	var minLat, maxLat, sumLat, avgLat, jitter float64
	if len(latencies) > 0 {
		minLat = latencies[0]
		maxLat = latencies[0]
		for _, l := range latencies {
			if l < minLat {
				minLat = l
			}
			if l > maxLat {
				maxLat = l
			}
			sumLat += l
		}
		avgLat = sumLat / float64(len(latencies))

		// Cálculo do jitter médio (RFC 1889 / RFC 3550: diferença média de latência entre amostras consecutivas)
		if len(latencies) > 1 {
			diffSum := 0.0
			for j := 1; j < len(latencies); j++ {
				diffSum += math.Abs(latencies[j] - latencies[j-1])
			}
			jitter = diffSum / float64(len(latencies)-1)
		}
	}

	res := &PacketLossResult{
		Target:       target,
		Sent:         count,
		Received:     received,
		LossPercent:  math.Round(lossPercent*10) / 10,
		JitterMs:     math.Round(jitter*100) / 100,
		MinLatencyMs: math.Round(minLat*100) / 100,
		AvgLatencyMs: math.Round(avgLat*100) / 100,
		MaxLatencyMs: math.Round(maxLat*100) / 100,
	}

	log.Printf("[MODULE:PACKET_LOSS] Resultado: Perda=%.1f%%, Jitter=%.2f ms, Latência Média=%.2f ms",
		res.LossPercent, res.JitterMs, res.AvgLatencyMs)
	return res
}

// RunDownloadModule executa o teste de velocidade de download
func RunDownloadModule(ctx context.Context, p *NetworkProbe, serverURL string, cfg SpeedtestDownloadConfig) float64 {
	if !cfg.Enabled || serverURL == "" || serverURL == "none" {
		return 0
	}
	bytesLimit := cfg.MaxBytes
	if bytesLimit <= 0 {
		bytesLimit = 10485760 // 10MB
	}

	log.Printf("[MODULE:DOWNLOAD] Iniciando teste de download (limite: %d bytes)...", bytesLimit)
	mbps := p.measureDownloadWithLimit(ctx, serverURL, bytesLimit)
	return math.Round(mbps*100) / 100
}

// RunUploadModule executa o teste de velocidade de upload
func RunUploadModule(ctx context.Context, p *NetworkProbe, serverURL string, cfg SpeedtestUploadConfig) float64 {
	if !cfg.Enabled || serverURL == "" || serverURL == "none" {
		return 0
	}
	bytesLimit := cfg.MaxBytes
	if bytesLimit <= 0 {
		bytesLimit = 5242880 // 5MB
	}

	log.Printf("[MODULE:UPLOAD] Iniciando teste de upload (limite: %d bytes)...", bytesLimit)
	mbps := p.measureUploadWithLimit(ctx, serverURL, bytesLimit)
	return math.Round(mbps*100) / 100
}

// RunDNSModule testa a resolução de nomes de domínio e tempo de resposta
func RunDNSModule(ctx context.Context, cfg DNSModuleConfig) *DNSResult {
	if !cfg.Enabled || len(cfg.Domains) == 0 {
		return nil
	}

	log.Printf("[MODULE:DNS] Testando resolução de %d domínios...", len(cfg.Domains))
	resolver := net.DefaultResolver

	var domainResults []DNSDomainResult
	var totalDuration float64
	successCount := 0

	for _, domain := range cfg.Domains {
		domain = strings.TrimSpace(domain)
		if domain == "" {
			continue
		}

		dnsCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		start := time.Now()
		ips, err := resolver.LookupHost(dnsCtx, domain)
		duration := float64(time.Since(start).Microseconds()) / 1000.0
		cancel()

		item := DNSDomainResult{
			Domain:     domain,
			DurationMs: math.Round(duration*100) / 100,
		}

		if err == nil && len(ips) > 0 {
			item.Success = true
			item.ResolvedIP = ips[0]
			totalDuration += duration
			successCount++
		} else if err != nil {
			item.Success = false
			item.Error = err.Error()
		}

		domainResults = append(domainResults, item)
	}

	avgMs := 0.0
	if successCount > 0 {
		avgMs = totalDuration / float64(successCount)
	}

	res := &DNSResult{
		Domains: domainResults,
		AvgMs:   math.Round(avgMs*100) / 100,
	}

	log.Printf("[MODULE:DNS] Concluído: %d/%d resolvidos, Média: %.2f ms", successCount, len(domainResults), res.AvgMs)
	return res
}

// RunHTTPCheckModule testa disponibilidade e latência de URL corporativa ou pública
func RunHTTPCheckModule(ctx context.Context, cfg HTTPCheckModuleConfig) *HTTPCheckResult {
	if !cfg.Enabled || strings.TrimSpace(cfg.URL) == "" {
		return nil
	}

	url := strings.TrimSpace(cfg.URL)
	expected := cfg.ExpectedStatus
	if expected <= 0 {
		expected = 200
	}

	log.Printf("[MODULE:HTTP_CHECK] Verificando status e latência de %s (esperado: %d)...", url, expected)

	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("redirecionamentos em excesso")
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return &HTTPCheckResult{
			URL:     url,
			Success: false,
			Error:   err.Error(),
		}
	}
	req.Header.Set("User-Agent", "UEPG-Net-Tools-Probe/2.0")

	start := time.Now()
	resp, err := client.Do(req)
	duration := float64(time.Since(start).Microseconds()) / 1000.0

	res := &HTTPCheckResult{
		URL:        url,
		DurationMs: math.Round(duration*100) / 100,
	}

	if err != nil {
		res.Success = false
		res.Error = err.Error()
		log.Printf("[MODULE:HTTP_CHECK] Falha ao conectar: %v", err)
		return res
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	res.StatusCode = resp.StatusCode
	res.Success = (resp.StatusCode == expected) || (resp.StatusCode >= 200 && resp.StatusCode < 400)

	log.Printf("[MODULE:HTTP_CHECK] Concluído: HTTP %d em %.1f ms (Sucesso: %v)", res.StatusCode, res.DurationMs, res.Success)
	return res
}

// RunTracerouteModule executa traceroute até o destino ou gateway
func RunTracerouteModule(ctx context.Context, cfg TracerouteModuleConfig) *TracerouteResult {
	if !cfg.Enabled {
		return nil
	}

	target := strings.TrimSpace(cfg.Target)
	if target == "" {
		target = "1.1.1.1"
	}
	maxHops := cfg.MaxHops
	if maxHops <= 0 || maxHops > 30 {
		maxHops = 15
	}

	log.Printf("[MODULE:TRACEROUTE] Executando traceroute para %s (máx saltos: %d)...", target, maxHops)

	res := &TracerouteResult{
		Target: target,
		Hops:   []TracerouteHop{},
	}

	// Executa tracert nativo do Windows (-d para não resolver DNS e ser mais rápido)
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "tracert", "-d", "-h", strconv.Itoa(maxHops), "-w", "500", target)
		hideWindow(cmd)
	} else {
		cmd = exec.CommandContext(ctx, "traceroute", "-n", "-m", strconv.Itoa(maxHops), "-w", "1", target)
	}

	out, err := cmd.Output()
	if err != nil && len(out) == 0 {
		log.Printf("[MODULE:TRACEROUTE] Erro ao executar traceroute: %v", err)
		res.Success = false
		return res
	}

	lines := strings.Split(string(out), "\n")
	hopRegex := regexp.MustCompile(`^\s*(\d+)\s+([<\d\.\*]+)\s*(?:ms)?\s+([<\d\.\*]+)\s*(?:ms)?\s+([<\d\.\*]+)\s*(?:ms)?\s+([0-9a-fA-F\.:]+|\*)`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		matches := hopRegex.FindStringSubmatch(line)
		if len(matches) >= 6 {
			hopNum, _ := strconv.Atoi(matches[1])
			ip := matches[5]

			latStr := strings.Trim(matches[2], "< ")
			lat, _ := strconv.ParseFloat(latStr, 64)

			res.Hops = append(res.Hops, TracerouteHop{
				HopNum:    hopNum,
				IP:        ip,
				LatencyMs: lat,
			})
		}
	}

	res.TotalHops = len(res.Hops)
	res.Success = res.TotalHops > 0

	log.Printf("[MODULE:TRACEROUTE] Concluído: %d saltos identificados", res.TotalHops)
	return res
}

// RunLLDPModule executa a descoberta de switch e VLAN
func RunLLDPModule(ctx context.Context, knownSwitch *SwitchInfo) SwitchInfo {
	var switchInfo SwitchInfo
	if knownSwitch != nil && knownSwitch.IsRealSwitch() {
		switchInfo = *knownSwitch
		log.Printf("[MODULE:LLDP] Switch corporativo preservado: %s (Porta %s, VLAN %d)", switchInfo.SwitchName, switchInfo.SwitchPort, switchInfo.VLAN)
		return switchInfo
	}

	log.Printf("[MODULE:LLDP] Buscando informações de switch via LLDP...")
	lldpCtx, cancelLldp := context.WithTimeout(ctx, 30*time.Second)
	switchInfo = DiscoverSwitchLLDP(lldpCtx)
	cancelLldp()

	if !switchInfo.IsRealSwitch() && knownSwitch != nil && knownSwitch.IsValid() {
		switchInfo = *knownSwitch
	}
	return switchInfo
}

// Métodos auxiliares para NetworkProbe parametrizado
func (p *NetworkProbe) pingViaExecWithParams(target string, count int, timeoutMs int) (float64, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("ping", "-n", strconv.Itoa(count), "-w", strconv.Itoa(timeoutMs), target)
		hideWindow(cmd)
	} else {
		cmd = exec.Command("ping", "-c", strconv.Itoa(count), "-W", strconv.Itoa(timeoutMs/1000+1), target)
	}

	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	outStr := string(output)
	avgRegex := regexp.MustCompile(`(?:M[eé]dia|Average)\s*=\s*(\d+)ms`)
	match := avgRegex.FindStringSubmatch(outStr)
	if len(match) > 1 {
		val, err := strconv.ParseFloat(match[1], 64)
		if err == nil && val >= 0 {
			return val, nil
		}
	}

	linuxRegex := regexp.MustCompile(`(?:rtt|round-trip)[^=]*=\s*[\d\.]+/([\d\.]+)/`)
	matchLinux := linuxRegex.FindStringSubmatch(outStr)
	if len(matchLinux) > 1 {
		val, err := strconv.ParseFloat(matchLinux[1], 64)
		if err == nil {
			return val, nil
		}
	}

	return 0, fmt.Errorf("não foi possível interpretar saída do ping")
}

func (p *NetworkProbe) measurePingFallback(target string) float64 {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(target, "80"), 2*time.Second)
	if err == nil {
		_ = conn.Close()
		return float64(time.Since(start).Microseconds()) / 1000.0
	}

	startDns := time.Now()
	connDns, errDns := net.DialTimeout("tcp", net.JoinHostPort(target, "53"), 2*time.Second)
	if errDns == nil {
		_ = connDns.Close()
		return float64(time.Since(startDns).Microseconds()) / 1000.0
	}
	return 0
}

func (p *NetworkProbe) measureDownloadWithLimit(ctx context.Context, serverURL string, maxBytes int64) float64 {
	url := fmt.Sprintf("%s/api/v1/speedtest/download", strings.TrimRight(serverURL, "/"))
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0
	}

	start := time.Now()
	resp, err := p.httpClient.Do(req)
	if err != nil {
		log.Printf("[PROBE] Falha no teste de download: %v", err)
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[PROBE] Servidor retornou HTTP %d no download", resp.StatusCode)
		return 0
	}

	limitedReader := io.LimitReader(resp.Body, maxBytes)
	bytesRead, err := io.Copy(io.Discard, limitedReader)
	duration := time.Since(start)

	if err != nil && err != io.EOF {
		log.Printf("[PROBE] Erro na leitura do stream de download: %v", err)
		return 0
	}

	if duration.Seconds() <= 0 || bytesRead == 0 {
		return 0
	}

	mbps := (float64(bytesRead) * 8.0) / (duration.Seconds() * 1000000.0)
	return mbps
}

func (p *NetworkProbe) measureUploadWithLimit(ctx context.Context, serverURL string, payloadSize int64) float64 {
	url := fmt.Sprintf("%s/api/v1/speedtest/upload", strings.TrimRight(serverURL, "/"))
	buf := make([]byte, payloadSize)
	_, _ = rand.Read(buf)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(buf))
	if err != nil {
		return 0
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	start := time.Now()
	resp, err := p.httpClient.Do(req)
	if err != nil {
		log.Printf("[PROBE] Falha no teste de upload: %v", err)
		return 0
	}
	defer resp.Body.Close()

	duration := time.Since(start)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		log.Printf("[PROBE] Servidor retornou HTTP %d no upload", resp.StatusCode)
		return 0
	}

	if duration.Seconds() <= 0 {
		return 0
	}

	mbps := (float64(payloadSize) * 8.0) / (duration.Seconds() * 1000000.0)
	return mbps
}
