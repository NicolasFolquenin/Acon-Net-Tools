# Módulos de Teste e Medição — Sonda Cliente

A sonda possui uma arquitetura modular de executores de teste (`client-probe/executors.go`). Cada teste é configurado independentemente via perfil de medição JSON e executado de forma controlada.

---

## 1. Módulos Disponíveis

### 1.1. Latência do Gateway Local (`ping_gateway`)
- **Tipo:** `ping`
- **Alvo Padrão:** `auto_gateway` (detectado dinamicamente via tabela de rotas do sistema operacional).
- **Mecanismo:** Dispara pacotes ICMP Echo Request nativos via comando do SO (`ping -n X -w Y` no Windows e `ping -c X -W Y` no Linux).
- **Fallback TCP:** Se o ICMP for bloqueado por firewall local, o executor tenta conexão TCP rápida (`pingViaTCP`) em portas conhecidas.

### 1.2. Perda de Pacotes (`packet_loss`)
- **Tipo:** `packet_loss`
- **Alvo:** Gateway local ou IP de referência institucional.
- **Funcionamento:** Dispara uma rajada de pacotes (padrão 5 a 10 amostras) e calcula a taxa exata de pacotes descartados:
  $$\text{Perda (\%)} = \frac{\text{Enviados} - \text{Recebidos}}{\text{Enviados}} \times 100$$
- **Parser Resiliente Windows & Linux:**
  - No Windows, faz a leitura direta dos campos `Enviados = (\d+)` e `Recebidos = (\d+)`, imune a quebras de linha em 80 colunas da saída de texto do prompt de comando.
  - No Linux, avalia `(\d+) packets transmitted, (\d+) received`.

### 1.3. Throughput de Download (`http_throughput_download`)
- **Tipo:** `http_throughput_download`
- **Alvo:** URL de arquivo de teste no Servidor Central (`/downloads/speedtest-payload` ou similar).
- **Funcionamento:** Efetua download de stream HTTP medindo o tempo decorrido e a quantidade de bytes recebidos, calculando a taxa média em Megabits por segundo (Mbps).

### 1.4. Throughput de Upload (`http_throughput_upload`)
- **Tipo:** `http_throughput_upload`
- **Alvo:** Endpoint de descarte do Servidor Central (`/api/v1/metrics/upload-sink`).
- **Funcionamento:** Transmite um payload sintético via `POST` HTTP multipart, medindo a velocidade real de transmissão do enlace em Mbps.

### 1.5. Resolução de DNS Institucional (`dns_lookup`)
- **Tipo:** `dns_lookup`
- **Alvo:** Nomes de domínio da instituição (ex: `uepg.br`, `sistemas.uepg.br`).
- **Funcionamento:** Mede o tempo em milissegundos para resposta dos servidores DNS corporativos configurados nas placas locais.

### 1.6. Descoberta de Topologia LLDP/CDP (`discovery_protocol`)
- **Tipo:** `discovery_protocol`
- **Funcionamento:**
  - No Linux: consulta o daemon `lldpd` ou faz sniffing em raw socket em nível de enlace (EtherType `0x88CC` para LLDP e `0x2000` para CDP da Cisco).
  - No Windows: escuta anúncios de switches gerenciáveis através dos adaptadores de rede físicos ativos.
  - Extrai: **Chassis ID / Nome do Switch**, **Port ID / Porta Conectada** (ex: `GE1/0/4`), **VLAN ID** e **Protocolo**.

