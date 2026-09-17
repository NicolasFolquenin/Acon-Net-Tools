# Sistema de Monitoramento de Rede Distribuído

Sistema completo de medição de throughput (Download e Upload) e latência de rede (Ping), distribuído em duas pastas autônomas:

- **`/client-probe`**: Aplicação Desktop em Go com **Wails v2** para Windows, com persistência local SQLite e interface moderna em Chart.js.
- **`/central-server`**: Servidor Central Coletor em Go 1.22+ para Linux/Debian com **MongoDB Time Series** e **Dashboard Web** responsivo com comparativos entre nós.

---

## 📂 Estrutura do Repositório

```
polling app/
├── client-probe/                     # 🖥️ Client Desktop Windows (Wails v2)
│   ├── .env                          # Variáveis de ambiente locais do client
│   ├── .env.example                  # Modelo de variáveis de ambiente
│   ├── frontend/                     # Interface do usuário (HTML, CSS, JS, Chart.js)
│   │   ├── dist/                     # Assets estáticos para embed no binário Go
│   │   ├── index.html                # UI principal com tema dark slate e glassmorphism
│   │   ├── style.css                 # Design System (slate, ciano #06b6d4, roxo #a855f7)
│   │   └── main.js                   # Chart.js (últimas 24 execuções), botões e eventos
│   ├── app.go                        # Métodos expostos ao Wails, background ticker 1h
│   ├── config.go                     # Leitura de .env e gravação de config.json
│   ├── config.json                   # Configurações do nó geradas/persistidas
│   ├── main.go                       # Entrypoint e configuração de janela Wails v2
│   ├── models.go                     # Contratos JSON e structs de banco
│   ├── probe.go                      # Medição de Ping (ICMP/TCP) e Throughput (Download/Upload)
│   ├── storage.go                    # Persistência SQLite (probe_local.db) e envio HTTPS
│   ├── wails.json                    # Manifesto de compilação Wails v2
│   ├── go.mod                        # Módulo Go do client
│   └── README.md                     # Guia de compilação e execução no Windows
│
└── central-server/                   # 🐧 Server Central Coletor (Linux Debian)
    ├── .env                          # Variáveis de ambiente do servidor e banco
    ├── .env.example                  # Modelo de variáveis de ambiente
    ├── public/                       # 🌐 Dashboard Web Central
    │   ├── index.html                # Painel de controle responsivo NOC
    │   ├── style.css                 # Tema Dark NOC (glassmorphism e grid responsivo)
    │   └── app.js                    # Seletor de nós, cartões de resumo e gráficos comparativos
    ├── config/                       # Carregador de .env e variáveis de ambiente
    │   └── config.go
    ├── db/                           # Driver oficial MongoDB v2 e Time Series setup
    │   └── mongodb.go
    ├── handlers/                     # Endpoints da API REST e middlewares de segurança
    │   ├── metrics.go                # POST /api/v1/metrics e GET /api/v1/dashboard/metrics
    │   ├── middleware.go             # Bearer Token Auth, CORS e Request Logger
    │   └── speedtest.go              # Endpoints para teste real de banda dos clients
    ├── models/                       # Structs MongoDB Time Series e DTOs de resposta
    │   └── metric.go
    ├── systemd/                      # Script de serviço para Debian Linux
    │   └── network-monitor.service
    ├── docker-compose.yml            # Go 1.22 Server + MongoDB 7 Time Series
    ├── Dockerfile                    # Build multi-stage para Linux Debian
    ├── main.go                       # Entrypoint do servidor HTTP e graceful shutdown
    ├── go.mod                        # Módulo Go do servidor central
    └── README.md                     # Guia de implantação Debian, cURL e MongoDB
```

---

## 📑 Contrato do Payload JSON (`POST /api/v1/metrics`)

Cabeçalho obrigatório:
```http
Authorization: Bearer <api_token>
Content-Type: application/json
```

Corpo da requisição:
```json
{
  "node_id": "minipc-dc-01",
  "location": "Guarapuava-DC1",
  "timestamp": "2026-09-11T15:00:00Z",
  "download_mbps": 850.5,
  "upload_mbps": 720.0,
  "ping_gateway": 1.2
}
```

---

## ⚡ Inicialização Rápida

### 1. Inicializando o Servidor Central (Linux/Debian ou Docker)
```bash
cd central-server
docker compose up -d --build
```
Acesse o Dashboard Web em: `http://localhost:8080`

Para gerar dados de demonstração imediatos de 24 horas:
```bash
curl -X POST http://localhost:8080/api/v1/seed
```

### 2. Inicializando o Client Desktop (Windows)
```powershell
cd client-probe
wails dev
```
Ou compilar executável `.exe` de produção:
```powershell
wails build -clean -platform windows/amd64
```
O executável será gerado em `client-probe/build/bin/probe-app.exe`.
