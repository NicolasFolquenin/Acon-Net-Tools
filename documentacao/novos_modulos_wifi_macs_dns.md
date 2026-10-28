# 📶 Novos Módulos de Rede: Wi-Fi, MACs e Multi-DNS

Este documento descreve os novos módulos introduzidos no ecossistema para suprir a necessidade de telemetria sem fio, identificação de múltiplos adaptadores e benchmark de latência DNS.

---

## 1. Módulo de Multi-Interfaces e Endereços MAC

### 1.1. Motivação
Em ambientes acadêmicos e corporativos, computadores frequentemente possuem mais de uma interface de rede:
- Uma porta cabeada Ethernet (RJ-45) ligada à bancada.
- Um adaptador Wi-Fi interno ou USB.
- Interfaces virtuais (VPN, WSL, Hyper-V, Docker).

Anteriormente, o sistema reportava apenas uma interface de rede genérica.

### 1.2. Implementação Técnica
- **Coleta no Client Probe ([`sysinfo.go`](../client-probe/sysinfo.go)):**
  - A função `collectNetworkDetails()` descobre a rota padrão ativa (`0.0.0.0/0`) no sistema operacional:
    - No Windows: consulta `Get-NetRoute -DestinationPrefix '0.0.0.0/0'` para determinar o nome da interface em uso e o IP do gateway padrão.
    - No Linux: consulta `ip route show default`.
  - Enumera todas as interfaces físicas e virtuais via `net.Interfaces()`.
  - Cada adaptador é classificado em:
    - `ethernet`: placas de rede cabeadas.
    - `wifi`: adaptadores sem fio 802.11.
    - `virtual`: adaptadores de software (loopback, vEthernet, tun, tap).
  - O adaptador com a rota default ativa é marcado com `IsPrimary: true`.
- **Contrato de Dados:**
  ```go
  type NetworkInterfaceDetail struct {
      Name         string   `json:"name"`
      Description  string   `json:"description,omitempty"`
      MACAddress   string   `json:"mac_address"`
      IPAddresses  []string `json:"ip_addresses"`
      Type         string   `json:"type"`       // ethernet, wifi, virtual
      IsPrimary    bool     `json:"is_primary"` // Rota padrao ativa
      IsUp         bool     `json:"is_up"`
      Gateway      string   `json:"gateway,omitempty"`
  }
  ```
- **Visualização no Dashboard:**
  - No modal **"Detalhes do Computador"**, uma tabela detalhada lista todas as placas encontradas, seus status e destaca a interface principal com badge dourado `★ Principal`.

---

## 2. Módulo de Medição Wi-Fi (`wifi_status`)

### 2.1. Métricas Capturadas
O módulo coleta informações detalhadas do enlace de rádio sem fio:
- **SSID**: Nome da rede Wi-Fi conectada (ex: `UEPG-WIFI`, `eduroam`).
- **BSSID**: Endereço MAC do Access Point (AP) ao qual a máquina está associada.
- **Potência do Sinal**:
  - Em porcentagem (`0%` a `100%`).
  - Em `dBm` calculado proporcionalmente (ex: `-55 dBm` para sinal excelente).
- **Padrão de Rádio**: Tecnologia 802.11 em uso (ex: `802.11ax` Wi-Fi 6, `802.11ac` Wi-Fi 5, `802.11n`).
- **Canal & Banda**: Canal de transmissão (ex: Canal 36, 5 GHz ou Canal 6, 2.4 GHz).
- **Taxas de Conexão**: Velocidade de negociação TX (transmissão) e RX (recepção) em Mbps.

### 2.2. Execução no Host
- **Windows ([`executors.go`](../client-probe/executors.go)):**
  - Utiliza o utilitário nativo `netsh wlan show interfaces`.
  - Converte as saídas localizadas do Windows em português e inglês.
- **Linux:**
  - Executa `nmcli -t -f active,ssid,bssid,signal,chan,rate dev wifi` ou comandos `iw`.
- **Tratamento Gracioso (Máquinas sem Wi-Fi):**
  - Se a máquina não possuir placa sem fio ou estiver com o rádio desativado, o executor retorna status `"skipped"`, indicando `"Não Aplicável"`.
  - Isso garante que máquinas puramente cabeadas operem normalmente sem registrar falsos erros ou alarmes no servidor central.

---

## 3. Módulo de Benchmark Multi-DNS (`multi_dns`)

### 3.1. Objetivo
Avaliar o tempo de resolução de nomes das sondas comparando o servidor DNS Institucional da UEPG contra grandes provedores públicos da internet.

### 3.2. Servidores Avaliados por Padrão
1. `177.101.22.40`: DNS Institucional UEPG (Campus Uvaranas / NTI).
2. `1.1.1.1`: Cloudflare Public DNS.
3. `8.8.8.8`: Google Public DNS.

### 3.3. Mecanismo de Teste
- O executor cria uma instância dedicada de `net.Resolver` para cada servidor DNS configurado.
- Utiliza conexão UDP direta na porta 53 com timeout estrito de 2.5 segundos:
  ```go
  resolver := &net.Resolver{
      PreferGo: true,
      Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
          d := net.Dialer{Timeout: 2500 * time.Millisecond}
          return d.DialContext(ctx, "udp", net.JoinHostPort(serverIP, "53"))
      },
  }
  ```
- O módulo resolve o domínio `uepg.br` (ou alvo configurado), mede a latência em milissegundos com precisão de frações e elege o servidor mais veloz (`fastest_server`).
- No dashboard web, os servidores são exibidos em uma tabela comparativa com indicadores visuais de performance:
  - 🟢 **Verde (< 30ms)**: Resolução rápida.
  - 🟡 **Amarelo (30ms - 80ms)**: Resolução aceitável.
  - 🔴 **Vermelho (> 80ms ou Falha)**: Lentidão ou timeout.
  - ⚡ **Tag Especial**: Destaca o servidor mais rápido do ciclo.
