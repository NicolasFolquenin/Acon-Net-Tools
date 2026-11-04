# UEPG Net Tools — Sistema Distribuído de Telemetria e Monitoramento de Rede

Sistema corporativo completo de monitoramento distribuído e telemetria de rede para o **Anel de Conectividade da UEPG**, composto por um **Servidor Central Coletor (Linux/VM)** com **Dashboard NOC** em tempo real e **Sondas Clientes Multi-plataforma (Linux e Windows)** com descoberta automática de topologia (LLDP) e perfis dinâmicos de medição.

---

## 📚 Documentação Técnica Modular

A documentação detalhada do sistema foi particionada em pastas temáticas dentro do diretório [`docs/`](docs/README.md):

- 🏛️ **[Arquitetura do Sistema](docs/arquitetura/visao_geral.md):** [Visão Geral](docs/arquitetura/visao_geral.md) | [Fluxo de Telemetria](docs/arquitetura/fluxo_telemetria.md) | [Segurança & Permissões](docs/arquitetura/seguranca_e_rede.md)
- 🐧 **[Servidor Central](docs/central-server/api_referencia.md):** [Referência da REST API](docs/central-server/api_referencia.md) | [Modelagem MongoDB](docs/central-server/banco_de_dados.md) | [Dashboard NOC Web](docs/central-server/dashboard_web.md)
- 🖥️ **[Sonda Cliente (Probe)](docs/client-probe/ciclo_de_execucao.md):** [Ciclo de Execução](docs/client-probe/ciclo_de_execucao.md) | [Módulos de Teste](docs/client-probe/modulos_de_teste.md) | [Auto-Update (v2.4.1)](docs/client-probe/auto_update.md) | [Modo Desktop Wails](docs/client-probe/modo_desktop_wails.md)
- 🚀 **[Deploy e Instalação](docs/deploy/servidor_docker.md):** [Servidor Docker na VM](docs/deploy/servidor_docker.md) | [Sonda Linux (systemd)](docs/deploy/sonda_linux.md) | [Sonda Windows (.exe)](docs/deploy/sonda_windows.md)
- 🎨 **[UX/UI & Design System](docs/ux-ui/diretrizes_visuais.md):** [Diretrizes Visuais](docs/ux-ui/diretrizes_visuais.md) | [Dashboard & Gráficos Chart.js](docs/ux-ui/dashboard_e_graficos.md)

Para o índice completo e navegação, consulte **[`docs/README.md`](docs/README.md)**.

---

## 🏛️ Componentes do Sistema

```
polling app/
├── central-server/                   # 🐧 Servidor Central Coletor (Linux / Docker / VM)
│   ├── deploy.sh                     # 🚀 Script de deploy automatizado em 1 comando
│   ├── docker-compose.yml            # Go 1.22 Server + MongoDB Time Series (AVX-compatible)
│   ├── Dockerfile                    # Build multi-stage otimizado
│   ├── handlers/                     # REST API, Telemetria, Auth, OIDC, LLDP, Perfis e Ping
│   ├── models/                       # Modelos de dados, Grupos de máquinas e Métricas
│   ├── public/                       # 🌐 Dashboard Web NOC em tempo real (Chart.js + Dark Theme)
│   │   └── downloads/                # Binários, instaladores e scripts servidos para os clients
│   └── systemd/                      # Script de serviço nativo systemd
│
└── client-probe/                     # 🖥️ Sonda Client Multi-plataforma (Linux & Windows)
    ├── cmd/                          # Entrypoints da aplicação CLI e Wails
    ├── frontend/                     # Interface Desktop Windows (Wails v2 + Glassmorphism)
    ├── app.go & main_cli.go          # Motores de execução periódica (Daemon & CLI)
    ├── executors.go & probe.go       # Executores: Ping ICMP/TCP, Throughput Download & Upload
    ├── lldp.go                       # Descoberta de Switch e Porta de Rede via LLDP
    ├── profile.go                    # Sincronização de perfis dinâmicos com o Servidor Central
    ├── storage.go                    # Buffer local SQLite resiliente a quedas de link
    └── build/                        # Scripts de compilação Linux amd64/arm64 e instalador Windows
```

---

## 🚀 1. Deploy do Servidor Central na VM (1 Comando)

Para instalar ou atualizar o Servidor Central em qualquer máquina virtual Linux (Debian 11/12, Ubuntu 20/22/24, CentOS, Rocky Linux):

```bash
# 1. Clonar o repositório
git clone https://git.uepg.br/anelconectividade/polling-app-acon.git
cd polling-app-acon/central-server

# (Ou via GitHub: git clone https://github.com/NicolasFolquenin/Acon-Net-Tools.git && cd Acon-Net-Tools/central-server)

# 2. Executar o script de deploy automatizado
chmod +x deploy.sh
sudo ./deploy.sh
```

### O que o `deploy.sh` faz por você:
- ✅ Verifica e instala automaticamente o **Docker** e o **Docker Compose** caso não estejam presentes.
- ✅ **Assistente no Terminal da VM:** Configura as variáveis (Porta, DNS, Domínio, Core, Senha) diretamente no terminal da VM no primeiro deploy, gravando tudo automaticamente no `.env` persistente.
- ✅ **Persistência Completa de Volumes:** Mantém `.env`, `./config` e `./mongo_data` preservados no host. Comandos como `docker compose down` e `up` não perdem dados nem pedem configs novamente.
- ✅ Customização total via `.env`: Porta, IP/Domínio do DNS, Core de rede e limites de teste podem ser alterados para qualquer instituição/empresa sem tocar em código.
- ✅ Garante compatibilidade de CPU em Hypervisors e VMs legadas (MongoDB Time Series 4.4 sem exigência de flags AVX).
- ✅ Inicia os containers em segundo plano e realiza teste de saúde no endpoint `/health`.
- ✅ Exibe na tela as URLs prontas de acesso e credenciais de administrador.

### Comandos de Gestão do Servidor:
```bash
./deploy.sh             # Deploy / Inicialização rápida (se já configurado, sobe direto sem perguntas)
./deploy.sh --wizard    # Abre o assistente interativo de configuração no terminal da VM
./deploy.sh --status    # Consulta o status dos containers e saúde do banco
./deploy.sh --logs      # Acompanha logs do servidor Go em tempo real
./deploy.sh --down      # Interrompe os containers com segurança (mantém os dados salvos)
./deploy.sh -y          # Executa deploy silencioso / não-interativo (para CI/CD)
```

---

## 🖥️ 2. Instalação das Sondas de Telemetria (Clients)

### 🐧 No Linux / Debian / Raspberry Pi (1 Comando via Terminal)
Execute o instalador oficial na máquina cliente:

```bash
curl -fsSL https://akvorado.acon.dev.uepg.br/downloads/install-probe.sh | sudo bash
```
*(Substitua a URL pelo IP/domínio do seu Servidor Central, se aplicável)*

- Registra o daemon `systemd` (`uepg-probe.service`) com auto-start no boot.
- Auto-registro do Hostname, IP, MAC e localização.
- Suporte a descoberta de porta de Switch via pacote `lldpd`.
- **Monitor com Animação WiFi:** Digite a qualquer momento no terminal:
  ```bash
  uepg-probe-monitor
  ```
  *(ou simplesmente `uepg-monitor`). Pressione `Ctrl + C` para sair a qualquer instante mantendo a sonda ativa em background.*

### 🪟 No Windows (Instalador Gráfico .exe)
1. Acesse o Dashboard Web do Servidor Central no navegador e clique em **Downloads**.
2. Baixe e execute o instalador `UEPG-Net-Tools-Instalador.exe`.
3. A sonda se integra à bandeja do sistema (System Tray), inicia com o Windows e armazena histórico local em SQLite com sincronização HTTPS.

---

## 📊 3. Funcionalidades do Dashboard Web NOC

- **Painel Comparativo de Throughput e Latência:** Gráficos interativos com séries temporais de Download, Upload, Ping ao Gateway e Ping ao Servidor Central.
- **Perfis Dinâmicos de Medição por Máquina:** Crie grupos e configure remotamente quais testes cada máquina executa (frequência, tamanho de pacotes, alvos).
- **Diagnóstico em Tempo Real:** Dispare testes de Ping ICMP instantâneos a partir do próprio Dashboard com visualizador de terminal e métricas instantâneas.
- **Topologia de Rede (LLDP):** Identificação automática do Switch e Porta física onde a máquina cliente está conectada.
- **Autenticação e Permissões:**
  - `admin` (senha `admin123`): Gerenciamento completo de nós, grupos, perfis e exclusão de máquinas inativas.
  - `operador` (senha `operador123`): Apenas leitura e visualização operacional.
  - Suporte nativo a Single Sign-On (SSO) institucional via Keycloak / OpenID Connect.

---

## 🔄 Repositórios e Ramificações (Git)

O projeto é versionado e mantido em sincronia contínua em dois repositórios oficiais:

- **GitLab Institucional UEPG:** `https://git.uepg.br/anelconectividade/polling-app-acon.git`
- **GitHub:** `https://github.com/NicolasFolquenin/Acon-Net-Tools.git`
- **Branch Principal de Integração:** `integration/unified-modular-system`
