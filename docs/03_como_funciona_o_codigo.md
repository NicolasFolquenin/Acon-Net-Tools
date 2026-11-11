# UEPG Net Tools — Documentação de Como Funciona o Código

> **Documentação Técnica de Arquitetura, Engenharia de Software e Fluxo de Dados**  
> **Projeto:** Sistema Distribuído de Telemetria e Diagnóstico de Rede  
> **Instituição:** Universidade Estadual de Ponta Grossa (UEPG) — Anel de Conectividade (ACON) / NTI  
> **Linguagem Principal:** Go (Golang) 1.22 | **Banco de Dados:** MongoDB TimeSeries & SQLite  
> **Frontend:** JavaScript Vanilla (SPA), HTML5, CSS3, Chart.js, Wails v2  
> **Versão:** v2.4.1  

---

## 1. Visão Geral da Arquitetura e Fluxo de Dados

O UEPG Net Tools foi projetado seguindo princípios de **alta resiliência**, **baixo consumo de recursos (CPU/RAM)** e **autonomia local nas pontas**.

```
+----------------------------------------------------------------------------------------------------+
|                                FLUXO GERAL DE DADOS E TELEMETRIA                                   |
+----------------------------------------------------------------------------------------------------+
|                                                                                                    |
| [ 1. Sonda Local: client-probe ]                                                                   |
|   ├── CollectSystemInfo(): Placas de rede, IPs, MACs, Gateway e Usuário ativo                     |
|   ├── DiscoverSwitchLLDP(): Escuta pacotes L2 na placa cabeada (Switch, Porta, VLAN)               |
|   ├── RunProfile(): Executa tarefas periódicas (Ping Gateway, Core, Loss, Throughput, Wi-Fi, DNS)  |
|   │                                                                                                |
|   ├── [ BUFFER LOCAL SQLite ] (storage.go)                                                         |
|   │     └── Salva medição localmente com status 'pending'                                          |
|   │                                                                                                |
|   └── [ MOTOR DE ENVIO HTTP ] (probe.go)                                                           |
|         ├── Tenta POST /api/v1/metrics no Central Server                                           |
|         ├── Se sucesso: marca como 'sent' no SQLite                                                |
|         └── Se falha de link: mantém no SQLite para reenvio ordenado no retorno da conexão         |
|                                                                                                    |
|                                   │  HTTP / JSON Payload                                           |
|                                   ▼                                                                |
|                                                                                                    |
| [ 2. Servidor Central: central-server ]                                                            |
|   ├── Ingestão (/api/v1/metrics - handlers/metrics.go)                                             |
|   │     ├── Valida integridade do JSON e autorização da sonda                                      |
|   │     ├── Atualiza cadastro da máquina (collection 'machines') com IPs, MACs, Switch e Usuários  |
|   │     └── Grava ponto temporal na coleção TimeSeries (collection 'metrics')                      |
|   │                                                                                                |
|   ├── Gestão de Perfis Dinâmicos (/api/v1/profiles - handlers/profiles.go)                         |
|   │     └── Devolve para a sonda as tarefas que ela deve executar no próximo ciclo                 |
|   │                                                                                                |
|   └── Diagnóstico de Ping Forçado (/api/v1/admin/ping - handlers/ping.go)                          |
|         ├── Descobre melhor IP ativo da máquina (Ethernet primária ou Wi-Fi)                        |
|         ├── Dispara ICMP executando ping nativo em sub-processo                                    |
|         └── Grava a medição no MongoDB TimeSeries e notifica o painel                              |
|                                                                                                    |
|                                   │  REST JSON / Cache em Memória                                  |
|                                   ▼                                                                |
|                                                                                                    |
| [ 3. Interface Web NOC: public/app.js ]                                                            |
|   ├── Single Page Application (SPA) modular com roteamento por Hash (#dashboard, #users, etc.)     |
|   ├── Renderização de Gráficos Incrementais com Chart.js                                           |
|   ├── Painel Técnico de Máquina Única com proteção de topologia física L2                          |
|   └── Atualização automática de dados a cada 10 segundos sem reload de página                      |
+----------------------------------------------------------------------------------------------------+
```

---

## 2. Funcionamento Detalhado da Sonda (`client-probe`)

O código-fonte da sonda reside no diretório [`client-probe/`](file:///c:/Users/João%20Ramos/Documents/Acon/polling-app-acon/client-probe).

### 2.1. Pontos de Entrada (`main.go` e `main_cli.go`)
- **Modo Desktop Windows (`main.go`):**  
  Utiliza o framework **Wails v2** para renderizar a interface gráfica nativa em HTML/CSS/JS empacotada no WebView2 do Windows. Instancia a struct `App` ([`app.go`](file:///c:/Users/João%20Ramos/Documents/Acon/polling-app-acon/client-probe/app.go)) e expõe métodos de backend para o frontend via IPC bridge (`wails:bind`).
- **Modo Daemon / CLI Linux (`main_cli.go`):**  
  Compilado sem dependências de GUI (tags `cli` ou compilação pura Linux). Roda como serviço de segundo plano `systemd`, executando o loop de polling e respondendo a sinais do sistema (`SIGINT`, `SIGTERM`).

---

### 2.2. O Ciclo de Execução Principal (`app.go` & `probe.go`)
1. **Inicialização do Sistema (`startup`):**
   - Inicializa o banco local SQLite embutido ([`storage.go`](file:///c:/Users/João%20Ramos/Documents/Acon/polling-app-acon/client-probe/storage.go)).
   - Executa `CollectSystemInfo()` para mapear o hardware imediatamente.
   - Restaura o último switch LLDP válido armazenado em cache no disco (desde que a máquina não esteja operando em Wi-Fi).
   - Inicia a goroutine de escuta LLDP em segundo plano (até 3 tentativas resilientes de 45–75 segundos para capturar os pacotes do switch).
   - Inicia a checagem periódica de atualização de software ([`auto_update.go`](file:///c:/Users/João%20Ramos/Documents/Acon/polling-app-acon/client-probe/auto_update.go)).
2. **Loop Periódico de Polling (`runScheduledLoop`):**
   - A cada intervalo configurado (padrão de 60 segundos), busca o perfil de testes do grupo no Servidor Central (`SyncProfile`).
   - Se houver personalização específica para a sonda no servidor, o perfil individual sobrepõe o padrão do grupo.
   - Executa a função `RunProfile()` ([`probe.go`](file:///c:/Users/João%20Ramos/Documents/Acon/polling-app-acon/client-probe/probe.go)).

---

### 2.3. Executores de Medição Modular (`executors.go`)
Todas as tarefas de teste implementam a interface `TaskExecutor`:
- **`ExecuteGatewayPingTask` e `ExecuteCorePingTask`:**  
  Executam 4 pacotes de ping ICMP usando subprocessos com timeout estrito ou sockets nativos. Calculam min/avg/max latency e taxa de perda.
- **`ExecuteThroughputTask`:**  
  Realiza download e upload de streams de bytes contra os endpoints `/api/v1/throughput/download` e `/api/v1/throughput/upload` do Central Server. Mede tempo decorrido em microssegundos e calcula Mbps reais.
- **`ExecuteMultiDNSTask`:**  
  Gera consultas `A` concorrentes contra o DNS interno da UEPG (`177.101.22.40`) e provedores públicos (`1.1.1.1`, `8.8.8.8`) medindo tempo de resolução em microssegundos.
- **`ExecuteWiFiStatusTask`:**  
  Mede status, potência (dBm e %), SSID, canal, frequência e taxa Tx/Rx:
  - **No Windows:** Faz o parsing de `netsh wlan show interfaces`.
    > **Atenção Técnica à Mitigação de Falso Positivo:**  
    > O código avalia `desconectado`/`disconnected` prioritariamente antes de checar `conectado`, pois a palavra em português *"desconectado"* contém a substring *"conectado"*. Além disso, valida obrigatoriamente se `res.SSID != ""` para impedir que adaptadores desassociados simulem uma conexão ativa.
  - **No Linux:** Faz o parsing de `nmcli -t -f active,ssid,bssid,signal,freq,chan dev wifi`.

---

### 2.4. Descoberta de Topologia Física L2 (`lldp.go`)
- **No Windows:** Abre a placa de rede cabeada em modo de escuta de baixo nível usando raw sockets / Npcap. Escuta o EtherType `0x88CC` (LLDP) e `0x2000` (Cisco CDP).
- **No Linux:** Consulta o daemon local `lldpd` via CLI estruturado (`lldpctl -f json`) ou sockets UNIX nativos.
- **Parsing de TLVs (Type-Length-Value):**  
  Extrai do pacote Ethernet:
  - `TLV 1 (Chassis ID):` MAC do switch.
  - `TLV 2 (Port ID):` Nome da porta física (ex: `GE1/0/4`).
  - `TLV 5 (System Name):` Nome do switch (ex: `ACONS0001`).
  - `TLV 127 (IEEE 802.1 / VLAN ID):` VLAN física de enlace (ex: `1101`).

---

### 2.5. Inventário de Hardware e Mitigação de Conexão (`sysinfo.go` e `models.go`)
- **Enumeração de Adaptadores:** `net.Interfaces()` combinado com chamadas WMI / PowerShell no Windows e `/sys/class/net/` no Linux.
- **Determinação de Placa Primária:** Avalia a tabela de rotas do sistema operacional para identificar qual placa responde pela rota `0.0.0.0/0` (Gateway Padrão).
- **Função `IsOnWiFi()` ([`models.go`](file:///c:/Users/João%20Ramos/Documents/Acon/polling-app-acon/client-probe/models.go)):**
  ```go
  func (s *SystemInfo) IsOnWiFi() bool {
      // 1. Se a interface primária ativa for Ethernet, está no cabo! Retorna false
      for _, iface := range s.Interfaces {
          if iface.IsPrimary && iface.IsUp {
              if iface.Type == "ethernet" || (!iface.IsWireless && !strings.Contains(...)) {
                  return false
              }
          }
      }
      // 2. Se a interface primária ativa for Wi-Fi, retorna true
      ...
  }
  ```
  Isso garante que se um computador de laboratório tiver uma placa Wi-Fi plugada, mas estiver navegando pelo cabo de rede, o sistema **preserva o switch LLDP e não o substitui por Wi-Fi**.

---

### 2.6. Inventário de Usuários Locais e Sessão Ativa
- **Windows:** Executa comandos PowerShell com saída forçada em `[Console]::OutputEncoding = [System.Text.Encoding]::UTF8`, coletando:
  - Usuário ativo atual via `qwinsta` e variáveis de ambiente da sessão.
  - Lista completa de contas locais via `Get-LocalUser`, identificando privilégios (Administrador vs Usuário Padrão), status da conta (Habilitada vs Desabilitada) e descrição.
- **Linux:** Lê `/etc/passwd` para inventariar contas com UID >= 1000 e consulta `who` / `loginctl` para identificar a sessão gráfica ativa.

---

### 2.7. Armazenamento Local e Resiliência Offline (`storage.go`)
Utiliza um banco **SQLite** embutido no arquivo local `probe_data.db`:
- **Tabela `metrics_buffer`:** Armazena payloads completos em formato JSON. Se o Central Server estiver indisponível, as métricas continuam sendo gravadas localmente.
- **Tabela `switch_cache`:** Salva o último switch físico válido descoberto via LLDP, permitindo persistir o ponto de rede mesmo que a sonda reinicie e o switch demore 60 segundos para anunciar o próximo quadro LLDP.

---

### 2.8. Atualização Automática Hot-Restart (`auto_update.go`)
- A cada 30 minutos, consulta `GET /api/v1/version` no Central Server.
- Se a versão remota for superior à versão local:
  1. Baixa o binário temporário na pasta de cache.
  2. Valida o hash criptográfico SHA256 do arquivo contra o fornecido pelo servidor.
  3. No Linux: substitui o binário em `/usr/local/bin/uepg-probe` e executa restart via `systemctl`.
  4. No Windows: cria um processo auxiliar que espera o encerramento do executável anterior, substitui o `.exe` e o reinicia automaticamente.

---

## 3. Funcionamento Detalhado do Servidor Central (`central-server`)

O código-fonte do servidor reside no diretório [`central-server/`](file:///c:/Users/João%20Ramos/Documents/Acon/polling-app-acon/central-server).

### 3.1. Roteamento e Inicialização (`main.go`)
Configura o roteador HTTP nativo de alta performance do Go, registrando os endpoints da REST API:
- `POST /api/v1/metrics` — Ingestão de telemetria da sonda.
- `GET /api/v1/metrics/latest` — Últimos dados para o Dashboard.
- `GET /api/v1/metrics/summaries` — Resumos em tempo real agregados por nó.
- `GET /api/v1/machines` — Cadastro de máquinas e inventário completo.
- `POST /api/v1/admin/ping` — Disparo de ping ICMP administrativo sob demanda.
- `GET /api/v1/profiles` & `POST /api/v1/groups` — Gestão de perfis e módulos de teste.
- `GET /downloads/*` — Servidor de arquivos estáticos de instalação e binários.
- `GET /` — Servidor do frontend Web do Dashboard NOC.

---

### 3.2. Ingestão de Telemetria (`handlers/metrics.go`)
Ao receber um payload da sonda:
1. Valida o schema JSON e o `NodeID`.
2. Extrai informações de rede, placas, switch LLDP, Wi-Fi e contas de usuários.
3. Atualiza o documento da máquina na coleção `machines` do MongoDB (upsert atômico), garantindo que as informações de hardware, usuários e switch fiquem registradas mesmo se a sonda desligar.
4. Insere o documento na coleção TimeSeries `metrics`.

---

### 3.3. Disparo Inteligente de Ping ICMP Administrativo (`handlers/ping.go`)
Quando o administrador clica em *"Forçar Ping"* no Dashboard:
1. O backend resolve o melhor IP da máquina via `resolveBestMachineIP()`:
   - Prioriza a interface que está com status `UP` e é a rota primária ou Wi-Fi ativo.
2. Executa o comando de ping no sistema operacional do servidor (com timeout controlado de 18 segundos).
3. **Fallback Automático:** Se o primeiro IP falhar (100% de perda) e a máquina possuir outras placas ativas (ex: o usuário tirou o cabo e está no Wi-Fi com outro IP), a função `findAlternativeIPs()` testa automaticamente o IP secundário.
4. **Gravação Direta no Banco (`recordPingMetric`):** Constrói um payload de métrica e o insere diretamente no MongoDB TimeSeries, garantindo que o ping forçado alimente os gráficos de latência e perda e a tabela em tempo real.

---

### 3.4. Camada de Dados MongoDB (`db/mongodb.go`)
- **Coleção TimeSeries `metrics`:**  
  Configurada com granularidade temporal por segundo, particionada por `node_id`. Oferece compressão nativa de dados e consultas por intervalo de data ultrarrápidas.
- **Coleção `machines`:**  
  Mantém o estado atual consolidado de cada computador (último IP, MACs, switch LLDP, grupo, VLAN, lista de usuários locais e customizações ativas).
- **Agregação em Memória (`computeNodeSummaries`):**  
  Calcula em milissegundos o resumo de todas as sondas da rede para renderizar os cards e tabelas do Dashboard com mínima sobrecarga de I/O de disco.

---

### 3.5. Frontend do Painel NOC (`public/app.js`)
Desenvolvido em **JavaScript Moderno Vanilla (SPA)** para máxima velocidade, leveza e compatibilidade:
- **Gestão de Gráficos:** Instâncias de `Chart.js` mantidas em memória com atualização incremental via `chart.update('none')`, evitando flickering visual.
- **Painel Técnico Embutido (`renderSingleMachineDetailsPanel`):**  
  Inspeciona as propriedades da máquina selecionada:
  - Protege a topologia de switch: se existir um switch físico válido (`ACONS0001`), o painel nunca o sobrepõe por Wi-Fi.
  - Renderiza o medidor de sinal Wi-Fi somente se a rede sem fio estiver genuinamente associada com SSID válido.
  - Monta a tabela de adaptadores de rede físicos destacando a placa com rota de Gateway.
- **Ciclo de Atualização:** Executa `loadDashboardData()` a cada 10 segundos, atualizando os componentes de tela de forma silenciosa e reativa.

