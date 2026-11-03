package main

import (
	"context"
	"log"
	"math"
	"net/http"
	"time"
)

// NetworkProbe é o serviço responsável por orquestrar a execução das tarefas de telemetria
type NetworkProbe struct {
	httpClient *http.Client
	executors  map[TaskType]TaskExecutor
}

// NewNetworkProbe cria uma nova instância de NetworkProbe com o registro padrão de executores
func NewNetworkProbe() *NetworkProbe {
	return &NetworkProbe{
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				DisableCompression:  true,
			},
		},
		executors: NewExecutorRegistry(),
	}
}

// RegisterExecutor permite registrar ou substituir um executor para um tipo de tarefa.
// Isso possibilita adicionar novos tipos de medição sem modificar o código do orquestrador.
func (p *NetworkProbe) RegisterExecutor(taskType TaskType, executor TaskExecutor) {
	if p.executors == nil {
		p.executors = make(map[TaskType]TaskExecutor)
	}
	p.executors[taskType] = executor
}

// RunProfile executa dinamicamente todas as tarefas ativas descritas no perfil de medição
func (p *NetworkProbe) RunProfile(ctx context.Context, prof *MeasurementProfile, cfg Config, knownSwitch *SwitchInfo, onProgress func(taskName string)) (*MetricPayload, error) {
	if prof == nil {
		prof = NewProfileManager().GetBuiltinFallback()
	}

	log.Printf("[PROBE] Iniciando ciclo de testes dinâmicos com perfil '%s' (%d tarefas cadastradas)...", prof.Version, len(prof.Tests))

	execCtx := &ExecutionContext{
		ServerURL:   cfg.ServerURL,
		KnownSwitch: knownSwitch,
		HTTPClient:  p.httpClient,
	}

	// Pré-carrega informações do host para alimentar alvos que necessitam dele (ex: auto_gateway)
	sysInfo := CollectSystemInfo()
	sysInfo.ClientVersion = CurrentClientVersion
	execCtx.SysInfo = &sysInfo

	metricsMap := make(map[string]interface{})
	var currentSwitch *SwitchInfo
	if knownSwitch != nil {
		currentSwitch = knownSwitch
	}

	var pingGateway float64
	var downloadMbps float64
	var uploadMbps float64

	for _, task := range prof.Tests {
		if !task.Enabled {
			continue
		}

		if onProgress != nil {
			onProgress(task.Name)
		}
		log.Printf("[PROBE] Executando tarefa [%s]: %s (tipo: %s)", task.ID, task.Name, task.Type)

		timeout := 30 * time.Second
		if task.TimeoutSeconds > 0 {
			timeout = time.Duration(task.TimeoutSeconds) * time.Second
		}
		taskCtx, cancel := context.WithTimeout(ctx, timeout)

		executor, registered := p.executors[task.Type]
		if !registered {
			log.Printf("[PROBE][AVISO] Tipo de tarefa desconhecido ignorado: %s (nenhum executor registrado)", task.Type)
			cancel()
			continue
		}

		result := executor.Execute(taskCtx, task, execCtx)
		metricsMap[task.ID] = result.Metric

		// Propaga dados laterais de volta ao contexto compartilhado
		if result.SwitchInfo != nil && (currentSwitch == nil || result.SwitchInfo.IsRealSwitch()) {
			currentSwitch = result.SwitchInfo
			execCtx.KnownSwitch = result.SwitchInfo
		}
		if result.SystemInfo != nil {
			execCtx.SysInfo = result.SystemInfo
		}

		// Preenche campos legados de compatibilidade com o dashboard existente
		switch task.Type {
		case TaskTypePing:
			if task.ID == "ping_gateway" || pingGateway == 0 {
				pingGateway = result.Metric.Value
			}
		case TaskTypeThroughputDownload:
			downloadMbps = result.Metric.Value
		case TaskTypeThroughputUpload:
			uploadMbps = result.Metric.Value
		}

		cancel()
	}

	// Identifica Grupo e VLAN correspondente
	var groupName string
	var vlan int
	if currentSwitch != nil {
		vlan = currentSwitch.VLAN
	}
	groupCfg := GetCurrentGroupConfig()
	if groupCfg.GroupName != "" {
		groupName = groupCfg.GroupName
	}

	payload := &MetricPayload{
		NodeID:         cfg.NodeID,
		Location:       cfg.Location,
		GroupName:      groupName,
		VLAN:           vlan,
		Timestamp:      time.Now().UTC().Truncate(time.Second),
		ProfileVersion: prof.Version,
		ClientVersion:  CurrentClientVersion,
		DownloadMbps:   math.Round(downloadMbps*100) / 100,
		UploadMbps:     math.Round(uploadMbps*100) / 100,
		PingGateway:    math.Round(pingGateway*100) / 100,
		Metrics:        metricsMap,
		SystemInfo:     execCtx.SysInfo,
		SwitchInfo:     currentSwitch,
	}

	return payload, nil
}

// MeasureAll executa os testes do perfil ativo ou fallback
func (p *NetworkProbe) MeasureAll(ctx context.Context, cfg Config, knownSwitch *SwitchInfo) (*MetricPayload, error) {
	defaultProf := NewProfileManager().GetBuiltinFallback()
	return p.RunProfile(ctx, defaultProf, cfg, knownSwitch, nil)
}
