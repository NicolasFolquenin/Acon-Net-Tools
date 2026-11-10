# UEPG Net Tools — O Que É e O Que Ele Faz

> **Documento Oficial de Visão Geral e Capacidades**  
> **Projeto:** Sistema Distribuído de Telemetria e Diagnóstico de Rede  
> **Instituição:** Universidade Estadual de Ponta Grossa (UEPG) — Anel de Conectividade (ACON) / NTI  
> **Versão:** v2.4.1  

---

## 1. O Que É o UEPG Net Tools?

O **UEPG Net Tools** é um ecossistema corporativo completo de monitoramento contínuo, diagnóstico ativo e telemetria distribuída de rede. Ele foi concebido para fornecer **visibilidade total em tempo real sobre a saúde e a qualidade da infraestrutura de comunicação** do Anel de Conectividade (ACON) e dos campi da Universidade Estadual de Ponta Grossa.

Em redes corporativas e acadêmicas de grande porte — com múltiplos blocos, laboratórios de informática, reitorias, hospitais universitários e servidores centrais —, os métodos tradicionais de monitoramento baseados em sondagem centralizada (como pingar máquinas a partir de um servidor) enfrentam limitações severas:
- Não capturam a percepção real da rede sob o ponto de vista da máquina cliente (a experiência do usuário na ponta).
- Falham em mapear a qual porta física de switch ou ponto de rede (jack) o computador está conectado.
- Não identificam quedas de sinal Wi-Fi, saturação de banda local ou problemas no DNS que afetam apenas um laboratório ou setor.

O **UEPG Net Tools resolve esse desafio invertendo a lógica tradicional**: ele distribui **sondas inteligentes e leves (Client-Probes)** nos computadores dos blocos e laboratórios. Essas sondas executam rotinas de testes periódicos e enviam os dados compilados para um **Servidor Central (Central Server)**, que consolida a telemetria em um **Dashboard Web em Tempo Real** para a equipe de redes (NOC/NTI).

```
+-----------------------------------------------------------------------------------+
|                            UEPG NET TOOLS - ECOSSISTEMA                           |
+-----------------------------------------------------------------------------------+
|                                                                                   |
|  [ SONDAS DISTRIBUÍDAS NAS PONTAS ]              [ NÚCLEO CENTRAL & GESTÃO ]      |
|                                                                                   |
|  +------------------------------+                +-----------------------------+  |
|  | Sonda Windows (Desktop/GUI)  |                | Servidor Central (Linux/VM) |  |
|  | - Testes periódicos          |   JSON / HTTPS | - API REST Go 1.22          |  |
|  | - Descoberta de Switch LLDP  | -------------> | - Banco TimeSeries MongoDB  |  |
|  | - Buffer local SQLite        |                | - Ingestão e Agregação      |  |
|  +------------------------------+                +-----------------------------+  |
|                                                                 |                 |
|  +------------------------------+                               v                 |
|  | Sonda Linux (Daemon systemd) |                +-----------------------------+  |
|  | - Execução contínua em boot  |   JSON / HTTPS | Dashboard Web NOC           |  |
|  | - Monitor interativo CLI     | -------------> | - Visão global de sondas    |  |
|  | - Perfil dinâmico remoto     |                | - Painel técnico por sonda  |  |
|  +------------------------------+                | - Ping ICMP forçado         |  |
|                                                  | - Topologia Switch & Wi-Fi  |  |
|                                                  +-----------------------------+  |
+-----------------------------------------------------------------------------------+
```

---

## 2. O Que o Net Tools Faz? (Funcionalidades Principais)

O sistema opera de forma modular e automatizada, cobrindo seis grandes pilares de monitoramento:

### 2.1. Métricas de Conectividade e Latência em Tempo Real
- **Ping para Gateway Local:** Mede o Round-Trip Time (RTT mínimo, médio e máximo) entre a sonda e o roteador/gateway padrão do setor, detectando gargalos no primeiro salto da rede.
- **Ping para o Core Institucional (NTI):** Mede a latência de trânsito até os roteadores centrais do anel de conectividade.
- **Perda de Pacotes (Packet Loss):** Monitora a integridade da transmissão (0% a 100% de perda), permitindo identificar cabos rompidos, conectores frouxos, interferências em enlaces de rádio ou portas danificadas antes mesmo do usuário abrir chamado.
- **Diagnóstico de Ping ICMP Forçado (Sob Demanda):** O administrador pode, a partir do Dashboard Central, disparar um teste de ping instantâneo contra qualquer computador cadastrado. O teste avalia a interface primária ativa, testa interfaces secundárias (Wi-Fi) se necessário e grava o resultado diretamente no banco de dados para alimentar os gráficos em tempo real.

### 2.2. Medição de Throughput Controlado (Download e Upload)
- Executa transferências HTTP/TCP segmentadas e controladas contra o servidor de testes do ACON.
- Calcula a taxa de transferência efetiva em Megabits por segundo (**Mbps**).
- Os testes são dimensionados para medir a capacidade real do link local sem sobrecarregar a banda externa de internet da universidade.

### 2.3. Descoberta Automática de Topologia Física L2 (Switch LLDP / CDP)
- **Escuta de Quadros LLDP (Link Layer Discovery Protocol / IEEE 802.1AB):** A sonda escuta passivamente a placa de rede cabeada para capturar os anúncios periódicos emitidos pelos switches corporativos (Cisco, HP, Huawei, Edgecore).
- **Mapeamento de Porta Física:** Identifica com precisão cirúrgica a porta do switch onde o cabo de rede está conectado (exemplo: `Switch: ACONS0001`, `Porta: GE1/0/4`).
- **Detecção de VLAN L2:** Extrai a VLAN física associada à porta do switch (exemplo: `VLAN 1101`), permitindo auditar se a máquina está na sub-rede correta.
- **Visualizador de Vizinhos de Switch:** No painel web, o administrador pode clicar em *"Ver Vizinhos Conectados neste Switch"* e listar instantaneamente todas as demais sondas que compartilham aquele mesmo equipamento de rede.
- **Invalidação Inteligente ao Desconectar:** Se o cabo de rede for desconectado e o usuário passar a navegar por Wi-Fi, a sonda limpa a informação de switch e notifica o servidor, evitando falsos registros de topologia cabeada.

### 2.4. Telemetria e Monitoramento de Redes Sem Fio (Wi-Fi)
- **Potência do Sinal:** Mede a intensidade do sinal em porcentagem (**%**) e em decibéis-miliwatts (**dBm**), categorizando em sinal Excelente, Bom, Regular ou Fraco.
- **Identificação da Rede:** Captura o nome da rede (**SSID**) e o endereço físico do rádio (**BSSID / MAC do Access Point**), permitindo saber a qual ponto de acesso específico a máquina está associada.
- **Canais e Frequências:** Identifica o canal de operação (ex: Canal 36, 149, etc.) e a faixa de frequência (**2.4 GHz, 5 GHz ou 6 GHz / Wi-Fi 6E/7**).
- **Taxas de Transmissão e Recepção (Tx/Rx Rates):** Reporta as velocidades negociadas pela placa sem fio em tempo real.
- **Proteção Contra Falsos Positivos:** O sistema diferencia com precisão se a máquina está usando Wi-Fi para navegar ou se está no cabo com o adaptador Wi-Fi desassociado em segundo plano.

### 2.5. Inventário de Placas de Rede, MACs e Usuários do Sistema
- **Inventário de Adaptadores de Rede:** Lista todas as placas de rede do equipamento (Ethernet, Wi-Fi, adaptadores virtuais, etc.), com status operacional (`Ativa/UP` ou `Inativa/DOWN`), endereços IPv4 atribuídos e endereços físicos MAC.
- **Identificação de Interface Primária:** Sinaliza claramente qual adaptador é o responsável pela rota padrão e tráfego de dados (Gateway Principal).
- **Rastreamento de Usuário Ativo:** Informa qual usuário do sistema operacional estava autenticado no exato instante em que o teste de telemetria foi disparado.
- **Inventário de Contas Locais Persistente:** Registra todas as contas de usuário do computador (ativas, inativas, privilégio de Administrador ou Padrão), permitindo que a equipe de suporte consulte a lista de usuários mesmo que a máquina esteja desligada ou offline.

### 2.6. Auditoria de Resolução DNS Institucional
- Testa a resolução de nomes contra o servidor DNS institucional da UEPG (`177.101.22.40`) e provedores públicos globais (Cloudflare `1.1.1.1`, Google `8.8.8.8`).
- Mede o tempo de resposta do DNS em milissegundos e alerta em caso de falha de resolução ou inconsistência de zonas internas.

---

## 3. Benefícios Práticos para a Instituição

| Para Quem | Benefício Prático |
| :--- | :--- |
| **Equipe de Redes (NOC / NTI)** | Visão centralizada de toda a malha em um único painel web. Alertas instantâneos de queda de enlace, identificação de saturação de switches e histórico de latência/perda em banco TimeSeries. |
| **Suporte Técnico nos Campi** | Localização física imediata do computador: o técnico sabe exatamente em qual switch e qual porta de rede o computador está conectado sem precisar ir ao rack de patch panels ou seguir cabos fisicamente. |
| **Gestão e Segurança** | Auditoria de contas de usuários locais nas máquinas laboratoriais e administrativas; histórico de quais usuários utilizaram cada equipamento. |
| **Usuários e Alunos** | Detecção proativa de degradação da rede antes de afetar aulas práticas, sistemas acadêmicos e serviços administrativos. |

---

## 4. Resumo da Arquitetura

O sistema é dividido em dois componentes principais independentes:

1. **`central-server` (Servidor Central):**
   - Desenvolvido em **Go 1.22**.
   - Banco de dados **MongoDB TimeSeries** para alta performance e compressão temporal de métricas.
   - Fornece a **REST API** para ingestão e controle.
   - Fornece o **Dashboard Web NOC** moderno com tema escuro institucional, gráficos Chart.js e painel técnico embutido.
   - Roda via Docker ou nativo em máquinas virtuais Linux Debian/Ubuntu.

2. **`client-probe` (Sonda Cliente):**
   - Desenvolvido em **Go 1.22**.
   - Disponível como **aplicativo Desktop Windows com GUI (Wails v2)** e como **daemon de segundo plano Linux (systemd)**.
   - Possui **Buffer Local em SQLite**: se o link de rede cair, a sonda guarda as medições no disco local e as envia automaticamente quando o link retornar, sem perda de dados históricos.
   - Suporte a **Auto-Update**: a sonda verifica periodicamente novas versões no Central Server, baixa o binário compilado, valida o checksum SHA256 e se auto-atualiza sem intervenção do usuário.
