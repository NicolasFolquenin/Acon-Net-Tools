# 🏗️ Arquitetura do Sistema Distribuído

O **UEPG Net Tools / Polling App Acon** é composto por uma arquitetura cliente-servidor desacoplada, projetada para alta disponibilidade, segurança de dados em rede acadêmica e resiliência a interrupções locais.

```
┌────────────────────────────────────────────────────────┐
│                   CENTRAL SERVER                       │
│  - REST API (Go 1.22)                                  │
│  - MongoDB Time-Series Collection                      │
│  - Autenticação JWT & Keycloak / OpenID Connect        │
│  - Dashboard Web NOC (Vanilla JS, CSS Moderno)         │
│  - Gestor de Perfis Dinâmicos & Auto-Update            │
└──────────────────────────┬─────────────────────────────┘
                           │ HTTPS / WSS / REST
        ┌──────────────────┴──────────────────┐
        ▼                                     ▼
┌───────────────────────────────┐ ┌───────────────────────────────┐
│     CLIENT PROBE (Windows)    │ │      CLIENT PROBE (Linux)     │
│ - Wails v2 (Desktop GUI)      │ │ - CLI / Daemon systemd        │
│ - SQLite local (offline cache)│ │ - SQLite local                │
│ - Windows Native Netsh/WLAN   │ │ - Linux Native nmcli/ip       │
│ - LLDP Discovery (WinPcap)    │ │ - LLDP Discovery (AF_PACKET)  │
│ - Engine Auto-Update          │ │ - Engine Auto-Update          │
└───────────────────────────────┘ └───────────────────────────────┘
```

---

## 1. Central Server

O **Servidor Central** centraliza a recepção de telemetria, orquestração de testes e controle de inventário:

- **Linguagem & Framework**: Go 1.22 com biblioteca padrão (`net/http`), garantindo alto throughput e baixo consumo de memória.
- **Banco de Dados Principal**: **MongoDB** utilizando *Time-Series Collections* para armazenar métricas com expiração e compactação otimizada por tempo.
- **Segurança & Controle de Acesso**:
  - Comunicação de sondas autenticada via **Bearer Token** institucional.
  - Painel administrativo com controle por perfis: `admin` (leitura, escrita, exclusão e criação de grupos) e `operator` (visualização).
  - Suporte a SSO corporativo via **OpenID Connect / Keycloak**.
- **Perfis Dinâmicos**: O servidor mantém em memória e banco o perfil de testes atribuído a cada nó ou grupo. Quando o operador altera um teste na interface, a sonda recebe o novo plano em seu próximo ciclo.
- **Distribuição de Binários & Auto-Update**: Endpoints dedicados (`/api/v1/download/client`, `/api/v1/download/version` e `/api/v1/download/binary/{platform}`) servem tanto o instalador interativo quanto binários para auto-patching silencioso.

---

## 2. Client Probe (Sonda de Telemetria)

O **Client Probe** é o agente instalado nas pontas da rede (computadores de laboratório, secretarias, reitoria, servidores locais ou Raspberry Pis):

### 2.1. Modo Desktop (Windows)
- Interface visual construída em **Wails v2**, renderizada com motor WebView nativo do Windows (sem o peso de navegadores embutidos como Chromium).
- Permite ao operador local visualizar em tempo real a velocidade de download/upload, perda de pacotes, endereço IP/MAC, porta do switch e status Wi-Fi.
- Inicia automaticamente com o Windows (configurável no instalador) e opera de forma transparente em segundo plano.

### 2.2. Modo CLI / Serviço (Linux)
- Binário compilado estaticamente para `linux/amd64` e `linux/arm64`.
- Funciona como daemon gerenciado por `systemd`, ideal para servidores Debian/Ubuntu ou Raspberry Pi.

### 2.3. Resiliência Offline (SQLite Local)
- Cada sonda possui um banco local **SQLite** (`probe_local.db`).
- Se a conexão com o Central Server ou com a internet for interrompida, as medições continuam sendo coletadas e armazenadas localmente.
- Assim que a conectividade é restabelecida, a sonda sincroniza os dados pendentes em lote.

---

## 3. Topologia e Descoberta de Switch (LLDP)
- As sondas escutam passivamente quadros **LLDP** (*Link Layer Discovery Protocol*) trafegados pela placa de rede.
- Capturam:
  - Nome do Switch corporativo (`Chassis ID` / `SysName`).
  - Número ou etiqueta da porta física (`Port ID`).
  - ID da VLAN local.
  - Modelo e fabricante do switch.
- Esses dados são enviados ao servidor central, permitindo que a equipe de infraestrutura saiba exatamente em qual switch e porta cada computador está conectado fisicamente.
