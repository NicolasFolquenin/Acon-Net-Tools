//go:build cli

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

const (
	cliVersion = "1.2.0-cli"
)

func main() {
	configPathFlag := flag.String("config", "", "Caminho do arquivo config.json (padrão: auto-detect)")
	serverURLFlag := flag.String("server", "", "URL do Servidor Central (ex: http://192.168.1.100:8080)")
	apiTokenFlag := flag.String("token", "", "Bearer Token para autenticação com o Servidor Central")
	intervalFlag := flag.Int("interval", 0, "Intervalo em minutos entre testes (0 para usar config)")
	onceFlag := flag.Bool("once", false, "Executa uma única medição, exibe o resumo e finaliza")
	versionFlag := flag.Bool("version", false, "Exibe a versão da sonda e encerra")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "UEPG Net Tools - Sonda Client CLI (%s - %s/%s)\n\n", cliVersion, runtime.GOOS, runtime.GOARCH)
		fmt.Fprintf(os.Stderr, "Uso:\n  uepg-probe [opções]\n\nOpções:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExemplos:\n")
		fmt.Fprintf(os.Stderr, "  uepg-probe -once\n")
		fmt.Fprintf(os.Stderr, "  uepg-probe -server http://10.0.0.1:8080 -token seu_token -once\n")
		fmt.Fprintf(os.Stderr, "  uepg-probe -config /etc/uepg-probe/config.json\n")
	}
	flag.Parse()

	if *versionFlag {
		fmt.Printf("uepg-probe versão %s (%s/%s)\n", cliVersion, runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	fmt.Println("==================================================================")
	fmt.Printf("   UEPG Net Tools - Sonda de Telemetria CLI v%s\n", cliVersion)
	fmt.Printf("   SO: %s | Arquitetura: %s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Println("==================================================================")

	// 1. Resolução do arquivo de configuração
	cfgPath := resolveConfigPath(*configPathFlag)
	cfgLoaded, err := LoadConfig(cfgPath)
	var cfg Config
	if err == nil && cfgLoaded != nil {
		cfg = *cfgLoaded
	} else {
		log.Printf("[CLI] Aviso: não foi possível carregar '%s' (%v). Usando padrões.", cfgPath, err)
		cfg = GetCurrentConfig()
	}

	// Sobrescreve com flags CLI se fornecidas
	if *serverURLFlag != "" {
		cfg.ServerURL = strings.TrimRight(*serverURLFlag, "/")
	}
	if *apiTokenFlag != "" {
		cfg.APIToken = strings.TrimSpace(*apiTokenFlag)
	}
	if *intervalFlag > 0 {
		cfg.IntervalMinutes = *intervalFlag
	}

	log.Printf("[CLI] Servidor Central: %s", cfg.ServerURL)
	if cfg.APIToken != "" {
		log.Printf("[CLI] Autenticação: Token configurado (%d caracteres)", len(cfg.APIToken))
	} else {
		log.Printf("[CLI] Autenticação: Nenhum token configurado")
	}

	// 2. Ajusta diretório do banco SQLite em Linux se for serviço
	setupStoragePath()

	storage, err := NewStorageService()
	if err != nil {
		log.Fatalf("[CLI] Erro fatal ao inicializar banco SQLite: %v", err)
	}
	defer storage.Close()

	probe := NewNetworkProbe()
	profileMgr := NewProfileManager()

	// 3. Coleta informações locais da máquina
	sysInfo := CollectSystemInfo()
	log.Printf("[CLI] Host: %s | IP: %s | MAC: %s | Gateway: %s", sysInfo.Hostname, sysInfo.IPAddress, sysInfo.MACAddress, sysInfo.Gateway)
	if sysInfo.OS != "" {
		log.Printf("[CLI] Sistema Operacional: %s", sysInfo.OS)
	}
	if sysInfo.CPU != "" {
		log.Printf("[CLI] Processador: %s | RAM: %.1f GB", sysInfo.CPU, sysInfo.RAMTotalGB)
	}

	// 4. Tenta recuperar switch de histórico ou escuta LLDP rápida
	var currentSwitch *SwitchInfo
	if validSw := storage.GetLastValidSwitchInfo(); validSw != nil {
		currentSwitch = validSw
		log.Printf("[CLI] Switch restaurado do cache local: %s (Porta %s, VLAN %d)", validSw.SwitchName, validSw.SwitchPort, validSw.VLAN)
	} else {
		log.Println("[CLI] Buscando informações de switch via LLDP (até 15s)...")
		lldpCtx, lldpCancel := context.WithTimeout(context.Background(), 15*time.Second)
		sw := DiscoverSwitchLLDP(lldpCtx)
		lldpCancel()
		if sw.IsRealSwitch() {
			currentSwitch = &sw
			log.Printf("[CLI] Switch identificado via LLDP: %s (Porta %s, VLAN %d)", sw.SwitchName, sw.SwitchPort, sw.VLAN)
		}
	}

	// Se o modo for apenas execução única (-once)
	if *onceFlag {
		log.Println("[CLI] Modo -once ativado: executando um ciclo completo de medição...")
		record, err := executeSingleRun(probe, profileMgr, storage, cfg, currentSwitch, &sysInfo)
		if err != nil {
			log.Fatalf("[CLI] Falha na execução da medição: %v", err)
		}
		printSummary(record)
		os.Exit(0)
	}

	// 5. Modo Daemon / Agendador Contínuo
	log.Println("[CLI] Modo Daemon ativado. Sonda pronta e aguardando horários programados.")
	if cfg.IsCentralSynced() {
		log.Println("[CLI] Sincronização: Centralizada (horários :00 e :30)")
	} else {
		log.Printf("[CLI] Sincronização: A cada %d minutos", cfg.IntervalMinutes)
	}

	// Executa uma medição inicial logo na inicialização para registrar a sonda no servidor
	go func() {
		log.Println("[CLI] Executando medição inicial de auto-registro...")
		time.Sleep(2 * time.Second)
		if rec, err := executeSingleRun(probe, profileMgr, storage, cfg, currentSwitch, &sysInfo); err == nil {
			if rec.SwitchInfo != nil && rec.SwitchInfo.IsRealSwitch() {
				currentSwitch = rec.SwitchInfo
			}
		}
	}()

	// Captura de sinais para finalização graciosa
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	workerCtx, cancelWorker := context.WithCancel(context.Background())
	defer cancelWorker()

	// Scheduler em segundo plano
	go func() {
		for {
			var next time.Time
			if cfg.IsCentralSynced() {
				next = getNextSyncSlot(time.Now())
			} else {
				interval := time.Duration(cfg.IntervalMinutes) * time.Minute
				if interval < 1*time.Minute {
					interval = 60 * time.Minute
				}
				next = time.Now().Add(interval)
			}

			waitDur := time.Until(next)
			if waitDur < 0 {
				waitDur = 50 * time.Millisecond
			}

			log.Printf("[CLI][SCHEDULER] Próxima medição agendada para: %s (em %v)", next.Format("15:04:05"), waitDur.Round(time.Second))

			timer := time.NewTimer(waitDur)
			select {
			case <-workerCtx.Done():
				timer.Stop()
				return
			case <-timer.C:
				log.Printf("[CLI][SCHEDULER] Horário atingido! Executando bateria de testes...")
				if rec, err := executeSingleRun(probe, profileMgr, storage, cfg, currentSwitch, &sysInfo); err == nil {
					if rec.SwitchInfo != nil && rec.SwitchInfo.IsRealSwitch() {
						currentSwitch = rec.SwitchInfo
					}
				}
			}
		}
	}()

	// Aguarda sinal de saída
	sig := <-sigChan
	log.Printf("[CLI] Sinal recebido (%v). Encerrando sonda com segurança...", sig)
	cancelWorker()
	log.Println("[CLI] Sonda finalizada.")
}

func executeSingleRun(probe *NetworkProbe, profileMgr *ProfileManager, storage *StorageService, cfg Config, sw *SwitchInfo, sys *SystemInfo) (*LocalMetricRecord, error) {
	// 1. Tenta atualizar o perfil dinâmico de testes junto ao servidor
	fetchCtx, cancelFetch := context.WithTimeout(context.Background(), 8*time.Second)
	bootstrapInfo := HostBootstrapInfo{
		Hostname:   sys.Hostname,
		IPAddress:  sys.IPAddress,
		MACAddress: sys.MACAddress,
	}
	if sw != nil {
		bootstrapInfo.SwitchName = sw.SwitchName
		bootstrapInfo.SwitchPort = sw.SwitchPort
		bootstrapInfo.VLAN = sw.VLAN
	}
	_, _ = profileMgr.FetchRemoteProfileForHost(fetchCtx, cfg.ServerURL, cfg.APIToken, bootstrapInfo)
	cancelFetch()

	activeProfile := profileMgr.GetActiveProfile()

	// 2. Executa a bateria de medições
	measureCtx, cancelMeasure := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancelMeasure()

	payload, err := probe.RunProfile(measureCtx, activeProfile, cfg, sw, func(taskName string) {
		log.Printf("[CLI][TASK] >> %s", taskName)
	})
	if err != nil {
		return nil, fmt.Errorf("erro no ciclo de testes: %w", err)
	}

	if payload.SwitchInfo != nil && payload.SwitchInfo.IsRealSwitch() {
		sw = payload.SwitchInfo
	}

	// 3. Envia para o servidor central
	sendCtx, cancelSend := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelSend()

	sendErr := storage.SendToServer(sendCtx, payload, cfg.ServerURL, cfg.APIToken)
	synced := true
	var errMsg string
	if sendErr != nil {
		synced = false
		errMsg = sendErr.Error()
		log.Printf("[CLI][SYNC] Servidor central indisponível ou erro: %v (salvo no SQLite para envio posterior)", sendErr)
	} else {
		log.Println("[CLI][SYNC] Métricas transmitidas com sucesso ao Servidor Central!")
	}

	// 4. Salva no banco SQLite local
	rec, err := storage.SaveMetric(payload, synced, errMsg)
	if err != nil {
		log.Printf("[CLI] Erro ao persistir no SQLite: %v", err)
	}

	// Se sincronizou com sucesso, tenta esvaziar fila de medições offline acumuladas
	if synced {
		go func() {
			syncCtx, syncCancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer syncCancel()
			if count := storage.SyncPendingMetrics(syncCtx, cfg.ServerURL, cfg.APIToken); count > 0 {
				log.Printf("[CLI][SYNC] %d medições pendentes anteriores foram sincronizadas com sucesso!", count)
			}
		}()
	}

	return rec, nil
}

func printSummary(rec *LocalMetricRecord) {
	if rec == nil {
		return
	}
	fmt.Println("\n========================= RESUMO DA MEDIÇÃO =========================")
	fmt.Printf(" Horário: %s\n", rec.Timestamp.Local().Format("02/01/2006 15:04:05"))
	fmt.Printf(" Status Central: ")
	if rec.Synced {
		fmt.Printf("\033[32mSINCRONIZADO COM SUCESSO\033[0m\n")
	} else {
		fmt.Printf("\033[33mOFFLINE (Gravado em cache local)\033[0m\n")
	}

	if rec.SystemInfo != nil {
		fmt.Printf(" Host: %s (%s) | IP: %s | Gateway: %s\n",
			rec.SystemInfo.Hostname, rec.SystemInfo.OS, rec.SystemInfo.IPAddress, rec.SystemInfo.Gateway)
	}
	if rec.SwitchInfo != nil && rec.SwitchInfo.IsRealSwitch() {
		fmt.Printf(" Switch: %s | Porta: %s | VLAN: %d\n",
			rec.SwitchInfo.SwitchName, rec.SwitchInfo.SwitchPort, rec.SwitchInfo.VLAN)
	}

	fmt.Println("---------------------------------------------------------------------")
	fmt.Println(" TAREFAS EXECUTADAS:")
	for taskID, rawRes := range rec.Metrics {
		if m, ok := rawRes.(map[string]interface{}); ok {
			val, _ := m["value"]
			unit, _ := m["unit"].(string)
			status, _ := m["status"].(string)
			details, _ := m["details"].(string)

			statusColor := "\033[32m[OK]\033[0m"
			if status == "error" {
				statusColor = "\033[31m[ERRO]\033[0m"
			} else if status == "warning" {
				statusColor = "\033[33m[AVISO]\033[0m"
			}

			fmt.Printf("  • %-18s %s: %v %s (%s)\n", taskID, statusColor, val, unit, details)
		}
	}
	fmt.Println("=====================================================================")
}

func resolveConfigPath(flagPath string) string {
	if flagPath != "" {
		return flagPath
	}
	// Procura em caminhos padrão
	candidates := []string{
		"config.json",
		"/etc/uepg-probe/config.json",
		filepath.Join(os.Getenv("HOME"), ".uepg-probe", "config.json"),
	}
	for _, p := range candidates {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}
	return "config.json"
}

func setupStoragePath() {
	if runtime.GOOS == "linux" {
		// Se a pasta /var/lib/uepg-probe existir, muda o diretório de trabalho para lá
		varDir := "/var/lib/uepg-probe"
		if fi, err := os.Stat(varDir); err == nil && fi.IsDir() {
			_ = os.Chdir(varDir)
		}
	}
}
