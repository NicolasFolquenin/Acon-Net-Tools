package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct gerencia o ciclo de vida do Wails e a coordenação de testes da sonda.
type App struct {
	ctx          context.Context
	cancelWorker context.CancelFunc
	storage      *StorageService
	probe        *NetworkProbe
	profileMgr   *ProfileManager
	isRunning    bool
	statusText   string
	activeTask   string
	lastRun      *time.Time
	nextRun      *time.Time
	lastMetric   *LocalMetricRecord
	currentPC    *SystemInfo
	currentSwitch *SwitchInfo
	mu           sync.Mutex
}

// NewApp cria uma nova instância de App
func NewApp() *App {
	return &App{
		probe:      NewNetworkProbe(),
		profileMgr: NewProfileManager(),
		statusText: "Inicializando sonda...",
	}
}

// startup é chamado quando o Wails inicializa a janela e o contexto
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	log.Println("[APP] Inicializando Desktop Probe...")

	// 1. Carrega configurações
	cfg, err := LoadConfig("")
	if err != nil {
		log.Printf("[APP] Erro ao carregar config.json: %v", err)
	}

	// 2. Inicializa SQLite local
	storage, err := NewStorageService()
	if err != nil {
		log.Fatalf("[APP] Erro critico ao inicializar SQLite: %v", err)
	}
	a.storage = storage

	// 3. Recupera o último registro do banco se houver
	history, _ := a.storage.GetRecentMetrics(1)
	if len(history) > 0 {
		a.lastMetric = &history[len(history)-1]
		a.lastRun = &a.lastMetric.Timestamp
		if a.lastMetric.SystemInfo != nil {
			a.currentPC = a.lastMetric.SystemInfo
		}
	}
	// Restaura o switch válido mais recente do histórico local para exibição instantânea
	if validSw := a.storage.GetLastValidSwitchInfo(); validSw != nil {
		a.currentSwitch = validSw
		log.Printf("[APP] Switch restaurado do cache local: %s (Porta %s)", validSw.SwitchName, validSw.SwitchPort)
	} else if len(history) > 0 && a.lastMetric.SwitchInfo != nil && a.lastMetric.SwitchInfo.IsRealSwitch() {
		a.currentSwitch = a.lastMetric.SwitchInfo
	} else {
		a.currentSwitch = &SwitchInfo{
			SwitchName: "Aguardando frame LLDP...",
			SwitchPort: "--",
			Protocol:   "LLDP",
		}
	}

	// 4. Coleta dados de hardware do PC imediatamente na inicialização
	go func() {
		pc := CollectSystemInfo()
		a.mu.Lock()
		a.currentPC = &pc
		a.mu.Unlock()
		a.emitStatus()

		// Busca inicial de LLDP em background (captura resiliente)
		// Faz até 3 tentativas (45s cada) até capturar o anúncio do switch corporativo (que emite a cada 30-60s)
		for attempt := 1; attempt <= 3; attempt++ {
			if a.currentSwitch != nil && a.currentSwitch.IsRealSwitch() {
				break
			}
			log.Printf("[APP] LLDP: Tentativa %d de escuta do switch corporativo (45s)...", attempt)
			timeout := 50 * time.Second
			if attempt == 1 {
				timeout = 75 * time.Second
			}
			lldpCtx, cancel := context.WithTimeout(context.Background(), timeout)
			sw := DiscoverSwitchLLDP(lldpCtx)
			cancel()
			if sw.IsRealSwitch() {
				a.mu.Lock()
				a.currentSwitch = &sw
				a.mu.Unlock()
				log.Printf("[APP] Switch corporativo identificado via LLDP: %s (Porta %s)", sw.SwitchName, sw.SwitchPort)
				a.emitStatus()
				break
			}
			time.Sleep(3 * time.Second)
		}
	}()

	var next time.Time
	if cfg.IsCentralSynced() {
		next = getNextSyncSlot(time.Now())
		a.statusText = "Sincronizado com o servidor central"
	} else {
		interval := time.Duration(cfg.IntervalMinutes) * time.Minute
		if interval < 1*time.Minute {
			interval = 60 * time.Minute
		}
		next = time.Now().Add(interval)
		a.statusText = fmt.Sprintf("Agendado (%dh)", int(interval.Hours()))
	}
	a.nextRun = &next

	// 4. Inicia goroutine de agendamento em segundo plano (:00 e :30 se sincronizado com servidor central)
	workerCtx, cancel := context.WithCancel(context.Background())
	a.cancelWorker = cancel
	go a.backgroundScheduler(workerCtx)

	// Sincroniza medições que porventura ficaram pendentes offline
	go func() {
		time.Sleep(2 * time.Second)
		syncCtx, syncCancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer syncCancel()
		if count := a.storage.SyncPendingMetrics(syncCtx, cfg.ServerURL, cfg.APIToken); count > 0 {
			a.emitStatus()
		}
	}()

	// Busca o perfil de medição mais recente do servidor central em segundo plano
	go func() {
		fetchCtx, cancelFetch := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelFetch()
		hostInfo := a.getHostBootstrapInfo()
		if prof, err := a.profileMgr.FetchRemoteProfileForHost(fetchCtx, cfg.ServerURL, cfg.APIToken, hostInfo); err == nil {
			log.Printf("[APP] Perfil dinâmico carregado do servidor para %s: versão %s (%d testes)", hostInfo.Hostname, prof.Version, len(prof.Tests))
			a.emitStatus()
		}
	}()

	log.Println("[APP] Probe inicializada com sucesso e pronta para operacao.")
}

// shutdown é chamado ao fechar o Wails
func (a *App) shutdown(ctx context.Context) {
	log.Println("[APP] Encerrando aplicacao...")
	if a.cancelWorker != nil {
		a.cancelWorker()
	}
	if a.storage != nil {
		a.storage.Close()
	}
}

// getNextSyncSlot calcula o próximo horário fixo (:00 ou :30) do relógio de parede
func getNextSyncSlot(now time.Time) time.Time {
	var target time.Time
	if now.Minute() < 30 {
		target = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 30, 0, 0, now.Location())
	} else {
		nextHour := now.Add(1 * time.Hour)
		target = time.Date(nextHour.Year(), nextHour.Month(), nextHour.Day(), nextHour.Hour(), 0, 0, 0, now.Location())
	}
	if !target.After(now) {
		target = target.Add(30 * time.Minute)
	}
	return target
}

// backgroundScheduler gerencia a execução periódica de testes.
// Quando configurado para sincronizar com o Servidor Central, todos os clientes disparam nos mesmos horários (:00 e :30).
func (a *App) backgroundScheduler(ctx context.Context) {
	for {
		cfg := GetCurrentConfig()
		var next time.Time
		var statusLabel string

		if cfg.IsCentralSynced() {
			next = getNextSyncSlot(time.Now())
			statusLabel = "Sincronizado com o servidor central"
		} else {
			interval := time.Duration(cfg.IntervalMinutes) * time.Minute
			if interval < 1*time.Minute {
				interval = 60 * time.Minute
			}
			next = time.Now().Add(interval)
			statusLabel = fmt.Sprintf("Agendado (%dh)", int(interval.Hours()))
		}

		a.mu.Lock()
		a.nextRun = &next
		if !a.isRunning {
			a.statusText = statusLabel
		}
		a.mu.Unlock()
		a.emitStatus()

		waitDuration := time.Until(next)
		if waitDuration < 0 {
			waitDuration = 50 * time.Millisecond
		}

		timer := time.NewTimer(waitDuration)
		select {
		case <-ctx.Done():
			timer.Stop()
			log.Println("[SCHEDULER] Goroutine em segundo plano finalizada.")
			return
		case <-timer.C:
			log.Printf("[SCHEDULER] Horário de medição atingido (%s)! Executando teste programado...", time.Now().Format("15:04:05"))
			_, _ = a.ExecuteTest("Agendado")
		}
	}
}

// RunTestNow é o método exposto ao frontend (Wails binding) para disparo manual
func (a *App) RunTestNow() (*LocalMetricRecord, error) {
	return a.ExecuteTest("Manual")
}

// ExecuteTest executa a rotina de medição, gravação local e envio central
func (a *App) ExecuteTest(triggerSource string) (*LocalMetricRecord, error) {
	a.mu.Lock()
	if a.isRunning {
		a.mu.Unlock()
		return nil, fmt.Errorf("um teste de rede ja esta em andamento")
	}
	a.isRunning = true
	a.statusText = fmt.Sprintf("Executando teste (%s)...", triggerSource)
	a.mu.Unlock()
	a.emitStatus()

	defer func() {
		a.mu.Lock()
		a.isRunning = false
		a.activeTask = ""
		cfg := GetCurrentConfig()
		if cfg.IsCentralSynced() {
			a.statusText = "Sincronizado com o servidor central"
		} else {
			interval := time.Duration(cfg.IntervalMinutes) * time.Minute
			a.statusText = fmt.Sprintf("Agendado (%dh)", int(interval.Hours()))
		}
		a.mu.Unlock()
		a.emitStatus()
	}()

	cfg := GetCurrentConfig()

	// 1. Tenta atualizar o perfil de testes antes de executar se o servidor estiver acessível
	fetchCtx, cancelFetch := context.WithTimeout(context.Background(), 5*time.Second)
	hostInfo := a.getHostBootstrapInfo()
	_, _ = a.profileMgr.FetchRemoteProfileForHost(fetchCtx, cfg.ServerURL, cfg.APIToken, hostInfo)
	cancelFetch()

	activeProfile := a.profileMgr.GetActiveProfile()

	// 2. Executa a bateria de medições dinâmicas definidas no perfil
	measureCtx, cancelMeasure := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancelMeasure()

	payload, err := a.probe.RunProfile(measureCtx, activeProfile, cfg, a.currentSwitch, func(taskName string) {
		a.mu.Lock()
		a.activeTask = taskName
		a.statusText = fmt.Sprintf("Executando: %s...", taskName)
		a.mu.Unlock()
		a.emitStatus()
	})
	if err != nil {
		log.Printf("[APP] Erro na medicao: %v", err)
		return nil, err
	}

	// Preserva o switch corporativo conhecido se a medição capturou apenas telefone ou vazio
	if payload.SwitchInfo != nil && payload.SwitchInfo.IsRealSwitch() {
		a.mu.Lock()
		a.currentSwitch = payload.SwitchInfo
		a.mu.Unlock()
	} else if a.currentSwitch != nil && a.currentSwitch.IsRealSwitch() {
		payload.SwitchInfo = a.currentSwitch
	} else if payload.SwitchInfo != nil && payload.SwitchInfo.IsValid() {
		a.mu.Lock()
		a.currentSwitch = payload.SwitchInfo
		a.mu.Unlock()
	} else if a.currentSwitch != nil && a.currentSwitch.IsValid() {
		payload.SwitchInfo = a.currentSwitch
	}

	// 2. Envia para o servidor central (POST com Bearer token)
	sendCtx, cancelSend := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelSend()

	sendErr := a.storage.SendToServer(sendCtx, payload, cfg.ServerURL, cfg.APIToken)
	var errMsg string
	synced := true
	if sendErr != nil {
		synced = false
		errMsg = sendErr.Error()
		log.Printf("[APP][SYNC] Falha ao enviar para o servidor central: %v", sendErr)
	} else {
		log.Printf("[APP][SYNC] Enviado com sucesso para o servidor central!")
	}

	// 3. Persiste no SQLite local
	record, err := a.storage.SaveMetric(payload, synced, errMsg)
	if err != nil {
		log.Printf("[APP] Erro ao gravar historico local: %v", err)
		return nil, err
	}

	// Se esta leitura sincronizou e há leituras pendentes antigas, sincroniza-as em background
	if synced {
		go func() {
			syncCtx, syncCancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer syncCancel()
			if count := a.storage.SyncPendingMetrics(syncCtx, cfg.ServerURL, cfg.APIToken); count > 0 {
				a.emitStatus()
			}
		}()
	}

	now := time.Now()
	a.mu.Lock()
	a.lastRun = &now
	a.lastMetric = record
	a.mu.Unlock()

	// Emite evento para a UI
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "metric_added", record)
	}

	return record, nil
}

// GetHistory retorna as últimas 24 medições para plotagem no Chart.js
func (a *App) GetHistory() ([]LocalMetricRecord, error) {
	if a.storage == nil {
		return []LocalMetricRecord{}, nil
	}
	return a.storage.GetRecentMetrics(24)
}

// GetHistoryPage retorna as medições paginadas (15 itens por página) com filtro por dia para a tabela
func (a *App) GetHistoryPage(dayFilter string, page int, pageSize int) (HistoryPageResult, error) {
	if a.storage == nil {
		return HistoryPageResult{Records: []LocalMetricRecord{}, Page: 1, PageSize: 15, TotalPages: 1}, nil
	}
	return a.storage.GetHistoryPage(dayFilter, page, pageSize)
}


// GetStatus retorna o estado geral da sonda
func (a *App) GetStatus() ProbeStatus {
	a.mu.Lock()
	defer a.mu.Unlock()

	var syncedCount, pendingCount int
	if a.storage != nil {
		syncedCount, pendingCount = a.storage.GetCounts()
	}

	serverStatus := "Online"
	if a.lastMetric != nil && !a.lastMetric.Synced {
		serverStatus = "Desconectado (offline)"
	}

	cfg := GetCurrentConfig()
	isSynced := cfg.IsCentralSynced()
	statusText := a.statusText
	if !a.isRunning && isSynced && (statusText == "" || strings.HasPrefix(statusText, "Agendado")) {
		statusText = "Sincronizado com o servidor central"
	}

	groupCfg := GetCurrentGroupConfig()
	groupName := groupCfg.GroupName
	currentVlan := 0
	if a.currentSwitch != nil {
		currentVlan = a.currentSwitch.VLAN
	}
	if a.lastMetric != nil && a.lastMetric.GroupName != "" {
		groupName = a.lastMetric.GroupName
	}

	return ProbeStatus{
		IsRunning:       a.isRunning,
		StatusText:      statusText,
		ActiveTask:      a.activeTask,
		GroupName:       groupName,
		CurrentVLAN:     currentVlan,
		LastRun:         a.lastRun,
		NextRun:         a.nextRun,
		LastMetric:      a.lastMetric,
		ServerStatus:    serverStatus,
		TotalSynced:     syncedCount,
		TotalPending:    pendingCount,
		CurrentPC:       a.currentPC,
		CurrentSwitch:   a.currentSwitch,
		CurrentProfile:  a.profileMgr.GetActiveProfile(),
		SyncWithCentral: isSynced,
	}
}

// GetActiveProfile expõe o perfil de medição ativo para a interface Wails
func (a *App) GetActiveProfile() *MeasurementProfile {
	if a.profileMgr == nil {
		return nil
	}
	return a.profileMgr.GetActiveProfile()
}

// RefreshProfile força a busca do perfil mais recente junto ao Servidor Central
func (a *App) RefreshProfile() (*MeasurementProfile, error) {
	if a.profileMgr == nil {
		return nil, fmt.Errorf("profile manager não inicializado")
	}
	cfg := GetCurrentConfig()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	hostInfo := a.getHostBootstrapInfo()
	prof, err := a.profileMgr.FetchRemoteProfileForHost(ctx, cfg.ServerURL, cfg.APIToken, hostInfo)
	if err == nil {
		a.emitStatus()
	}
	return prof, err
}

// GetConfig expõe a configuração para a UI
func (a *App) GetConfig() Config {
	return GetCurrentConfig()
}

// SaveConfig atualiza as configurações a partir da UI
func (a *App) SaveConfig(cfg Config) error {
	err := SaveConfig(cfg)
	if err == nil {
		a.mu.Lock()
		if a.cancelWorker != nil {
			a.cancelWorker()
		}
		workerCtx, cancel := context.WithCancel(context.Background())
		a.cancelWorker = cancel
		go a.backgroundScheduler(workerCtx)
		a.mu.Unlock()

		go func() {
			syncCtx, syncCancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer syncCancel()
			if count := a.storage.SyncPendingMetrics(syncCtx, cfg.ServerURL, cfg.APIToken); count > 0 {
				a.emitStatus()
			}
		}()
	}
	return err
}

func (a *App) emitStatus() {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "status_updated", a.GetStatus())
	}
}

// getHostBootstrapInfo coleta os dados locais do host e switch para identificação e auto-registro
func (a *App) getHostBootstrapInfo() HostBootstrapInfo {
	sys := CollectSystemInfo()
	hostname := GetSystemHostname()
	info := HostBootstrapInfo{
		Hostname:   hostname,
		IPAddress:  sys.IPAddress,
		MACAddress: sys.MACAddress,
	}

	a.mu.Lock()
	sw := a.currentSwitch
	a.mu.Unlock()

	if sw != nil {
		info.VLAN = sw.VLAN
		info.SwitchName = sw.SwitchName
		info.SwitchPort = sw.SwitchPort
	}
	return info
}

