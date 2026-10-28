# Central Server Coletor de Telemetria de Rede (Linux / VM)

Servidor central coletor de telemetria de alto desempenho escrito em **Go 1.22+**, integrado ao **MongoDB** com coleções **Time Series** nativas, **Dashboard Web NOC** responsivo com comparativos em tempo real, perfis dinâmicos de teste por máquina e suporte a autenticação local e institucional (Keycloak / OIDC).

---

## 🚀 Deploy Rápido na VM (1 Comando)

Para implantar o Servidor Central em qualquer máquina virtual ou servidor físico Linux (Debian, Ubuntu, CentOS, Rocky), basta clonar o repositório e executar o script de deploy automatizado:

```bash
git clone https://git.uepg.br/anelconectividade/polling-app-acon.git
cd polling-app-acon/central-server
chmod +x deploy.sh
sudo ./deploy.sh
```

*(Ou usando o repositório GitHub: `git clone https://github.com/NicolasFolquenin/Acon-Net-Tools.git && cd Acon-Net-Tools/central-server && sudo ./deploy.sh`)*

### O que o `deploy.sh` faz automaticamente:
1. **Detecção e Instalação do Docker:** Se o Docker e o Docker Compose não estiverem instalados, oferece instalação automática com o script oficial.
2. **Configuração de Ambiente (`.env`):** Detecta o IP local da VM, sugere a URL pública e configura a porta (padrão `8085`).
3. **Compatibilidade Ampla de CPU para VMs:** O banco de dados utiliza imagem MongoDB adaptada para rodar perfeitamente em Hypervisors (Proxmox, VMware, ESXi) mesmo sem instruções AVX.
4. **Construção e Execução dos Containers:** Sobe o servidor Go e o MongoDB em segundo plano via Docker Compose.
5. **Verificação de Saúde (Healthcheck):** Aguarda a inicialização e valida o endpoint `/health`.
6. **Exibição do Resumo:** Mostra URLs de acesso, credenciais padrão e comandos de administração.

---

## ⚙️ Opções do Script de Deploy

O script suporta flags para automação e manutenção:

```bash
# Deploy totalmente automatizado (modo não-interativo, aceita todos os padrões)
sudo ./deploy.sh -y

# Acompanhar logs do servidor em tempo real
./deploy.sh --logs

# Consultar status dos containers
./deploy.sh --status

# Parar os serviços com segurança (mantendo os dados do MongoDB preservados)
./deploy.sh --down

# Deploy nativo sem Docker (compilação direta em Go + serviço systemd)
sudo ./deploy.sh --native
```

---

## 🏛️ Arquitetura do Sistema

- **Backend:** Go 1.22+ com driver oficial `go.mongodb.org/mongo-driver/v2/mongo`.
- **Database:** MongoDB com coleção Time Series (`network_central.metrics`):
  - `timeField`: `timestamp`
  - `metaField`: `metadata` (`node_id`, `location`, `ip_address`, `mac_address`)
  - `granularity`: `hours`
- **Dashboard Web NOC:** Servido diretamente em `GET /` a partir de `./public` (HTML5, Vanilla JS, Chart.js, Glassmorphism dark theme).
- **Gerenciamento Dinâmico de Sondas:**
  - Auto-registro no primeiro heartbeat.
  - Alocação em Grupos de Máquinas (ex: *Default/Bootstrap*, *Laboratórios*, *Roteadores de Borda*).
  - Perfis de tarefas dinâmicas distribuídos via JSON (`/api/v1/probes/profile/{hostname}`).
- **Mecanismos de Autenticação:**
  - Bearer Token HTTP para envio de telemetria das sondas (`API_TOKEN`).
  - Sessão com Cookies HttpOnly para o Dashboard Web (`admin` / `operador`).
  - Single Sign-On (SSO) Institucional via OpenID Connect (Keycloak).

---

## 🔑 Credenciais Padrão do Dashboard

Ao acessar o Dashboard em `http://<IP_DA_VM>:8085` ou no seu domínio configurado:

| Usuário | Senha Padrão | Nível de Acesso |
| :--- | :--- | :--- |
| **admin** | `admin123` | Administrador total (gerencia perfis, grupos, módulos e remove nós) |
| **operador** | `operador123` | Leitura (visualização de gráficos, histórico e status) |

> [!IMPORTANT]
> Altere as senhas padrão no arquivo `.env` para ambientes de produção.

---

## 📡 Instalação das Sondas Clientes a partir do Servidor

Uma vez que o Servidor Central esteja no ar, ele passa a distribuir automaticamente os instaladores:

### 🐧 Sonda Linux / Debian (1 Comando no Terminal)
```bash
curl -fsSL http://<IP_OU_DOMINIO>:8085/downloads/install-probe.sh | sudo bash
```
- Instala a sonda como serviço `systemd` (`uepg-probe.service`) que roda 24/7.
- Acompanha o comando CLI **`uepg-probe-monitor`** (ou `uepg-monitor`) para reabrir a animação matemática ASCII de ondas WiFi a qualquer momento no terminal.

### 🪟 Sonda Windows (Instalador Gráfico .exe)
Baixe o instalador compilado diretamente pelo Dashboard Web em **Downloads** ou acesse:
```
http://<IP_OU_DOMINIO>:8085/api/v1/download/client
```

---

## 🛠️ Variáveis de Ambiente (`.env`)

Todas as variáveis do sistema são configuráveis centralizadamente pelo arquivo `.env`. Ao alterar uma variável no `.env`, ela se propaga automaticamente para os perfis das sondas, catálogo de módulos e interface do Dashboard:

```ini
# Porta e Endereço de Acesso
PORT=8085
PUBLIC_SERVER_URL=https://akvorado.acon.dev.uepg.br

# Banco de Dados MongoDB
MONGO_URI=mongodb://mongo:27017
MONGO_DB=network_central
MONGO_COLLECTION=metrics
API_TOKEN=super-secret-probe-token-2026
STATIC_DIR=/app/public
QUERY_HOURS_LIMIT=24

# 🌐 Alvos de Medição, Rede e DNS (Customizáveis para qualquer instituição/empresa)
DNS_SERVER_TARGET=177.101.22.40               # IP do servidor DNS principal (ex: 8.8.8.8, 1.1.1.1)
DNS_DOMAIN_TARGET=uepg.br                     # Domínio para resolução de teste (ex: google.com, suaempresa.com)
DNS_SERVER_NAME=DNS Institucional             # Nome exibido no Dashboard e relatórios
PING_CORE_TARGET=177.101.22.40                # IP ou Hostname do roteador central/backbone
PING_CORE_NAME=Core NTI (Anel)                # Rótulo da métrica do Core
DEFAULT_PING_TARGET=auto_gateway              # Alvo de ping local ('auto_gateway' detecta o gateway do nó)
SPEEDTEST_DOWNLOAD_MB=5                       # Volume do teste de throughput de download (MB)
SPEEDTEST_UPLOAD_MB=2                         # Volume do teste de throughput de upload (MB)

# Autenticação Local do Dashboard
AUTH_ADMIN_USER=admin
AUTH_ADMIN_PASSWORD=admin123
AUTH_VIEWER_USER=operador
AUTH_VIEWER_PASSWORD=operador123
SESSION_SECRET=super-secret-session-key-change-in-prod-2026

# Autenticação SSO (Opcional - Keycloak)
OIDC_ENABLED=false
OIDC_ISSUER_URL=https://sso.acon.dev.uepg.br/realms/uepg
OIDC_CLIENT_ID=net-tools
OIDC_CLIENT_SECRET=sua-chave-secreta
OIDC_REDIRECT_URL=https://akvorado.acon.dev.uepg.br/api/v1/auth/oidc/callback
```

---

## 📡 Endpoints da API REST

| Método | Endpoint | Descrição |
| :--- | :--- | :--- |
| `GET` | `/health` | Healthcheck do servidor (retorna status HTTP 200) |
| `POST` | `/api/v1/metrics` | Ingestão de medições enviadas pelas sondas (Bearer Token) |
| `GET` | `/api/v1/dashboard/metrics` | Consulta séries temporais para gráficos do Dashboard |
| `GET` | `/api/v1/probes/profile/{hostname}` | Perfil dinâmico de tarefas da máquina |
| `POST` | `/api/v1/probes/profile/{hostname}` | Atualização de perfil da máquina (Admin) |
| `GET` | `/api/v1/speedtest/download` | Payload binário para teste de banda de Download |
| `POST` | `/api/v1/speedtest/upload` | Sink descartável para teste de banda de Upload |
| `POST` | `/api/v1/auth/login` | Login no Dashboard (admin / operador) |
| `POST` | `/api/v1/auth/logout` | Encerramento de sessão |
