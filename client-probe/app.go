package main

import (
	"context"
	"fmt"
	"log"
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
	isRunning    bool
	statusText   string
	lastRun      *time.Time
	nextRun      *time.Time
	lastMetric    *LocalMetricRecord
	currentPC     *SystemInfo
	currentSwitch *SwitchInfo
	mu            sync.Mutex
}

// NewApp cria uma nova instância de App
func NewApp() *App {
	return &App{
		probe:      NewNetworkProbe(),
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
		if a.lastMetric.SwitchInfo != nil {
			a.currentSwitch = a.lastMetric.SwitchInfo
		}
	}

	// 4. Coleta dados de hardware do PC imediatamente na inicialização
	go func() {
		pc := CollectSystemInfo()
		a.mu.Lock()
		a.currentPC = &pc
		a.mu.Unlock()
		a.emitStatus()

		// Busca inicial de LLDP em background
		lldpCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		sw := DiscoverSwitchLLDP(lldpCtx)
		a.mu.Lock()
		a.currentSwitch = &sw
		a.mu.Unlock()
		a.emitStatus()
	}()

	interval := time.Duration(cfg.IntervalMinutes) * time.Minute
	if interval < 1*time.Minute {
		interval = 60 * time.Minute
	}
	next := time.Now().Add(interval)
	a.nextRun = &next
	a.statusText = fmt.Sprintf("Agendado (%dh)", int(interval.Hours()))

	// 4. Inicia goroutine de agendamento em segundo plano (a cada 1 hora)
	workerCtx, cancel := context.WithCancel(context.Background())
	a.cancelWorker = cancel
	go a.backgroundScheduler(workerCtx, interval)

	// Sincroniza medições que porventura ficaram pendentes offline
	go func() {
		time.Sleep(2 * time.Second)
		syncCtx, syncCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer syncCancel()
		if count := a.storage.SyncPendingMetrics(syncCtx, cfg.ServerURL, cfg.APIToken); count > 0 {
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

// backgroundScheduler roda em background disparando testes no intervalo configurado (1 hora)
func (a *App) backgroundScheduler(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[SCHEDULER] Goroutine em segundo plano finalizada.")
			return
		case <-ticker.C:
			log.Println("[SCHEDULER] Intervalo atingido! Executando teste programado...")
			_, _ = a.ExecuteTest("Agendado")
			next := time.Now().Add(interval)
			a.mu.Lock()
			a.nextRun = &next
			a.mu.Unlock()
			a.emitStatus()
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
		cfg := GetCurrentConfig()
		interval := time.Duration(cfg.IntervalMinutes) * time.Minute
		a.statusText = fmt.Sprintf("Agendado (%dh)", int(interval.Hours()))
		a.mu.Unlock()
		a.emitStatus()
	}()

	cfg := GetCurrentConfig()

	// 1. Executa medição de rede (Ping, Download, Upload)
	measureCtx, cancelMeasure := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancelMeasure()

	payload, err := a.probe.MeasureAll(measureCtx, cfg)
	if err != nil {
		log.Printf("[APP] Erro na medicao: %v", err)
		return nil, err
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
			syncCtx, syncCancel := context.WithTimeout(context.Background(), 20*time.Second)
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

	return ProbeStatus{
		IsRunning:     a.isRunning,
		StatusText:    a.statusText,
		LastRun:       a.lastRun,
		NextRun:       a.nextRun,
		LastMetric:    a.lastMetric,
		ServerStatus:  serverStatus,
		TotalSynced:   syncedCount,
		TotalPending:  pendingCount,
		CurrentPC:     a.currentPC,
		CurrentSwitch: a.currentSwitch,
	}
}

// GetConfig expõe a configuração para a UI
func (a *App) GetConfig() Config {
	return GetCurrentConfig()
}

// SaveConfig atualiza as configurações a partir da UI
func (a *App) SaveConfig(cfg Config) error {
	return SaveConfig(cfg)
}

func (a *App) emitStatus() {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "status_updated", a.GetStatus())
	}
}
