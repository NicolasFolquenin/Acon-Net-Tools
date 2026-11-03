# Visão Geral da Arquitetura

O **UEPG Net Tools** é um ecossistema distribuído concebido para monitorar com precisão a qualidade de enlace, desempenho de tráfego e topologia física dos pontos de rede no Anel de Conectividade da Universidade Estadual de Ponta Grossa (UEPG).

```
+--------------------------------------------------------------------------------+
|                                CENTRAL SERVER                                  |
|                                                                                |
|  +--------------------+     +---------------------+     +-------------------+  |
|  |   Dashboard NOC    |     |   Go REST Server    |     |      MongoDB      |  |
|  |   (HTML5/CSS/JS)   |<--->|      (API v1)       |<--->|    Time Series    |  |
|  +--------------------+     +---------------------+     +-------------------+  |
|                                        ^                                       |
+----------------------------------------|---------------------------------------+
                                         | HTTPS / REST (Auth Bearer Token)
                                         v
+--------------------------------------------------------------------------------+
|                             CLIENT PROBES (Sondas)                             |
|                                                                                |
|  +-------------------------------------+   +--------------------------------+  |
|  | Linux Probe (Daemon systemd)        |   | Windows Probe (Wails v2 GUI)   |  |
|  | - Engine Go + lldpd                 |   | - Engine Go + WebView2         |  |
|  | - Buffer SQLite local               |   | - Buffer SQLite local          |  |
|  | - CLI Terminal Monitor              |   | - Auto-Update em disco         |  |
|  +-------------------------------------+   +--------------------------------+  |
+--------------------------------------------------------------------------------+
```

---

## 1. Componentes Principais

### Servidor Central (`central-server`)
- **Linguagem & Runtime:** Go 1.22+ operando em container Docker ou nativo Linux.
- **Banco de Dados:** MongoDB com suporte a séries temporais (otimizado com MongoDB 4.4 para total compatibilidade com CPUs virtuais legadas sem instruções AVX).
- **Interface Web:** Dashboard NOC em HTML5 moderno com design system dual-theme (escuro e claro), gráficos Chart.js responsivos e SSE/Polling para dados em tempo real.
- **Servidor de Downloads:** Repositório HTTP interno que entrega instaladores de sondas, scripts de bootstrap e binários compilados para auto-atualização.

### Sonda Cliente (`client-probe`)
- **Linguagem & Runtime:** Go 1.22+ com suporte a compilação híbrida:
  - **Modo CLI / Daemon:** Executado no Linux sob controle do `systemd`, ideal para Raspberry Pi, Mini PCs ou servidores de borda.
  - **Modo Desktop Windows:** Aplicação gráfica empacotada com Wails v2 (WebView2 nativo do Windows), com painel local de diagnóstico e histórico SQLite.
- **Armazenamento Resiliente:** Banco de dados SQLite local (`probe_local.db`) que armazena os testes em caso de queda de link ou inacessibilidade temporária do Servidor Central.
- **Descoberta de Topologia (LLDP):** Módulo de escuta passiva de pacotes LLDP/CDP para identificar a marca do switch, nome de host corporativo, porta física de enlace e VLAN.

---

## 2. Tecnologias Utilizadas

| Camada | Tecnologia | Função Principal |
| :--- | :--- | :--- |
| **Backend Central** | Go (Golang) standard library + Chi Router | Roteamento rápido de API REST e agregação de telemetria |
| **Banco de Dados** | MongoDB (4.4 Time Series) | Armazenamento de métricas, nós registrados e perfis de teste |
| **Frontend Central** | Vanilla JavaScript (ES6+), CSS Grid/Flexbox, Chart.js 4.4 | Dashboard de NOC com alta performance e sem dependências pesadas |
| **Sonda Core** | Go 1.22, go-sqlite3 | Execução de testes de rede e persistência local resiliente |
| **Sonda GUI** | Wails v2, HTML5/CSS3, WebView2 | Aplicação gráfica para estações desktop Windows |
| **Topologia** | LLDP (Link Layer Discovery Protocol), raw sockets / lldpd | Descoberta física de switch, porta e VLAN |

