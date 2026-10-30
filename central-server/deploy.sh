#!/usr/bin/env bash
# ==============================================================================
# UEPG Net Tools - Script de Deploy Automatizado do Servidor Central (Linux / VM)
# Permite subir todo o ambiente (Go Server + MongoDB Time Series) em minutos.
# ==============================================================================
set -e

# Garante execução no diretório do script (central-server/)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Definição de cores ANSI para saída formatada
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# Variáveis e Modo de Execução
NON_INTERACTIVE=false
MODE="docker" # "docker" ou "native"
ACTION="deploy"

# Parse de Argumentos CLI
for arg in "$@"; do
  case "$arg" in
    -y|--yes|--non-interactive)
      NON_INTERACTIVE=true
      ;;
    -n|--native)
      MODE="native"
      ;;
    --logs)
      ACTION="logs"
      ;;
    --status)
      ACTION="status"
      ;;
    --down|--stop)
      ACTION="down"
      ;;
    --wizard)
      ACTION="wizard"
      ;;
    -h|--help)
      echo -e "${CYAN}${BOLD}UEPG Net Tools - Deploy do Servidor Central${NC}"
      echo -e "Uso: ${YELLOW}./deploy.sh [OPÇÕES]${NC}\n"
      echo -e "Opções:"
      echo -e "  ${GREEN}-y, --yes${NC}             Modo não-interativo (aceita todos os valores padrão automaticamente)"
      echo -e "  ${GREEN}-n, --native${NC}          Executar deploy nativo em Go + Systemd (sem Docker)"
      echo -e "  ${GREEN}--wizard${NC}              Executar o Assistente Interativo de Configuração do .env"
      echo -e "  ${GREEN}--status${NC}              Exibe status dos serviços em execução"
      echo -e "  ${GREEN}--logs${NC}                Acompanha logs em tempo real dos containers"
      echo -e "  ${GREEN}--down${NC}                Para os serviços com segurança (preserva dados do MongoDB)"
      echo -e "  ${GREEN}-h, --help${NC}            Exibe esta mensagem de ajuda"
      exit 0
      ;;
  esac
done

# Wrapper para executar comandos com sudo se não for root
SUDO=""
if [ "$EUID" -ne 0 ]; then
  if command -v sudo >/dev/null 2>&1; then
    SUDO="sudo"
  else
    echo -e "${RED}[ERRO] Este script requer privilégios de root ou o utilitário 'sudo' instalado.${NC}"
    exit 1
  fi
fi

# Banner visual inicial
echo -e "${BLUE}================================================================================${NC}"
echo -e "${CYAN}${BOLD}       UEPG Net Tools • Deploy Automatizado do Servidor Central (VM)          ${NC}"
echo -e "${BLUE}================================================================================${NC}\n"

# Detecção do utilitário Docker Compose
detect_docker_compose() {
  if docker compose version >/dev/null 2>&1; then
    DOCKER_COMPOSE="docker compose"
  elif docker-compose version >/dev/null 2>&1; then
    DOCKER_COMPOSE="docker-compose"
  elif command -v $SUDO >/dev/null 2>&1 && $SUDO docker compose version >/dev/null 2>&1; then
    DOCKER_COMPOSE="$SUDO docker compose"
  elif command -v $SUDO >/dev/null 2>&1 && $SUDO docker-compose version >/dev/null 2>&1; then
    DOCKER_COMPOSE="$SUDO docker-compose"
  else
    DOCKER_COMPOSE=""
  fi
}

detect_docker_compose

# Ações rápidas (--status, --logs, --down)
if [ "$ACTION" = "status" ]; then
  if [ -n "$DOCKER_COMPOSE" ]; then
    echo -e "${YELLOW}[+] Status dos containers Docker:${NC}"
    $DOCKER_COMPOSE ps
  fi
  if systemctl is-active --quiet network-monitor 2>/dev/null; then
    echo -e "${GREEN}[+] Serviço nativo systemd 'network-monitor' ativo:${NC}"
    systemctl status network-monitor --no-pager
  fi
  exit 0
fi

if [ "$ACTION" = "logs" ]; then
  if [ -n "$DOCKER_COMPOSE" ]; then
    $DOCKER_COMPOSE logs -f
  else
    journalctl -u network-monitor -f
  fi
  exit 0
fi

if [ "$ACTION" = "down" ]; then
  echo -e "${YELLOW}[+] Parando serviços do Servidor Central com segurança...${NC}"
  if [ -n "$DOCKER_COMPOSE" ]; then
    $DOCKER_COMPOSE down --remove-orphans
    echo -e "${GREEN}[✓] Containers parados. Dados em ./mongo_data foram preservados.${NC}"
  fi
  if systemctl is-active --quiet network-monitor 2>/dev/null; then
    $SUDO systemctl stop network-monitor
    echo -e "${GREEN}[✓] Serviço systemd network-monitor parado.${NC}"
  fi
  exit 0
fi

# ==============================================================================
# 1. VERIFICAÇÃO E INSTALAÇÃO DE DEPENDÊNCIAS (DOCKER)
# ==============================================================================
if [ "$MODE" = "docker" ]; then
  echo -e "${GREEN}[1/5] Verificando Docker e Docker Compose...${NC}"

  if ! command -v docker >/dev/null 2>&1; then
    echo -e "${YELLOW}[!] Docker não foi detectado no sistema.${NC}"
    if [ "$NON_INTERACTIVE" = false ]; then
      echo -e "Deseja instalar o Docker automaticamente via script oficial? (${GREEN}S/n${NC}): "
      read -rp "> " INSTALL_DOCKER_CONFIRM < /dev/tty || INSTALL_DOCKER_CONFIRM="S"
    else
      INSTALL_DOCKER_CONFIRM="S"
    fi

    if [[ "$INSTALL_DOCKER_CONFIRM" =~ ^[Nn] ]]; then
      echo -e "${RED}[ERRO] O Docker é necessário para o deploy padrão. Instale o Docker ou use './deploy.sh --native'.${NC}"
      exit 1
    fi

    echo -e "${GREEN}[+] Instalando Docker via script oficial...${NC}"
    curl -fsSL https://get.docker.com | $SUDO sh
    $SUDO systemctl enable --now docker

    # Adiciona usuário atual ao grupo docker para dispensar sudo futuramente
    if [ -n "$SUDO_USER" ]; then
      $SUDO usermod -aG docker "$SUDO_USER" 2>/dev/null || true
    fi
  fi

  detect_docker_compose
  if [ -z "$DOCKER_COMPOSE" ]; then
    echo -e "${YELLOW}[+] Instalando docker-compose-plugin...${NC}"
    if command -v apt-get >/dev/null 2>&1; then
      $SUDO apt-get update && $SUDO apt-get install -y docker-compose-plugin
    elif command -v yum >/dev/null 2>&1; then
      $SUDO yum install -y docker-compose-plugin
    fi
    detect_docker_compose
  fi

  if [ -z "$DOCKER_COMPOSE" ]; then
    echo -e "${RED}[ERRO] Não foi possível localizar ou instalar o Docker Compose.${NC}"
    exit 1
  fi
  echo -e "${GREEN}[✓] Docker e Docker Compose prontos (${CYAN}$($DOCKER_COMPOSE version)${GREEN})${NC}\n"
fi

# ==============================================================================
# 2. PREPARAÇÃO DO ARQUIVO DE CONFIGURAÇÃO (.env)
# ==============================================================================
echo -e "${GREEN}[2/5] Configurando variáveis de ambiente (.env)...${NC}"

if [ ! -f ".env" ]; then
  if [ -f ".env.example" ]; then
    echo -e "${YELLOW}[!] .env não encontrado. Copiando de .env.example...${NC}"
    cp .env.example .env
  else
    echo -e "${RED}[ERRO] Arquivo .env.example não encontrado.${NC}"
    exit 1
  fi
fi

# Detecção do IP local da VM
LOCAL_IP=$(ip -4 route get 1.1.1.1 2>/dev/null | awk '{print $7}' | tr -d '\n')
if [ -z "$LOCAL_IP" ]; then
  LOCAL_IP=$(hostname -I 2>/dev/null | awk '{print $1}')
fi
[ -z "$LOCAL_IP" ] && LOCAL_IP="127.0.0.1"

# Lê configurações atuais do .env
CURRENT_PORT=$(grep -E '^PORT=' .env 2>/dev/null | cut -d'=' -f2 | tr -d ' "' || echo "8085")
[ -z "$CURRENT_PORT" ] && CURRENT_PORT="8085"

CURRENT_DNS_SERVER=$(grep -E '^DNS_SERVER_TARGET=' .env 2>/dev/null | cut -d'=' -f2 | tr -d ' "' || echo "1.1.1.1")
[ -z "$CURRENT_DNS_SERVER" ] && CURRENT_DNS_SERVER="1.1.1.1"

CURRENT_DNS_DOMAIN=$(grep -E '^DNS_DOMAIN_TARGET=' .env 2>/dev/null | cut -d'=' -f2 | tr -d ' "' || echo "google.com")
[ -z "$CURRENT_DNS_DOMAIN" ] && CURRENT_DNS_DOMAIN="google.com"

CURRENT_PING_CORE=$(grep -E '^PING_CORE_TARGET=' .env 2>/dev/null | cut -d'=' -f2 | tr -d ' "' || echo "1.1.1.1")
[ -z "$CURRENT_PING_CORE" ] && CURRENT_PING_CORE="1.1.1.1"

CURRENT_PUBLIC_URL=$(grep -E '^PUBLIC_SERVER_URL=' .env 2>/dev/null | cut -d'=' -f2- | tr -d ' "' || true)
SUGGESTED_PUBLIC_URL="$CURRENT_PUBLIC_URL"
if [ -z "$SUGGESTED_PUBLIC_URL" ] || [ "$SUGGESTED_PUBLIC_URL" = "http://localhost:8080" ] || [ "$SUGGESTED_PUBLIC_URL" = "http://localhost:8085" ]; then
  SUGGESTED_PUBLIC_URL="http://${LOCAL_IP}:${CURRENT_PORT}"
fi

# Configuração interativa (Assistente de Instalação no Terminal)
if [ "$NON_INTERACTIVE" = false ] && [ -c /dev/tty ]; then
  echo -e "${CYAN}┌────────────────────────────────────────────────────────────────────────┐${NC}"
  echo -e "${CYAN}│${BOLD}   🛠️  ASSISTENTE INTERATIVO DE CONFIGURAÇÃO (SETUP WIZARD)            ${NC}${CYAN}│${NC}"
  echo -e "${CYAN}│   Pressione [ENTER] para aceitar os valores sugeridos entre colchetes. │${NC}"
  echo -e "${CYAN}└────────────────────────────────────────────────────────────────────────┘${NC}\n"

  echo -e "1. Porta TCP onde o servidor responderá [Padrão: ${CYAN}${CURRENT_PORT}${NC}]:"
  read -rp "   Porta [${CURRENT_PORT}]: " USER_PORT < /dev/tty
  TARGET_PORT="${USER_PORT:-$CURRENT_PORT}"

  echo -e "\n2. URL Pública ou Domínio de Acesso das sondas [Padrão: ${CYAN}${SUGGESTED_PUBLIC_URL}${NC}]:"
  read -rp "   URL Pública [${SUGGESTED_PUBLIC_URL}]: " USER_URL < /dev/tty
  TARGET_URL="${USER_URL:-$SUGGESTED_PUBLIC_URL}"

  echo -e "\n3. IP do Servidor DNS Principal para testes [Padrão: ${CYAN}${CURRENT_DNS_SERVER}${NC}]:"
  read -rp "   DNS Server IP [${CURRENT_DNS_SERVER}]: " USER_DNS < /dev/tty
  TARGET_DNS="${USER_DNS:-$CURRENT_DNS_SERVER}"

  echo -e "\n4. Domínio para testes de resolução DNS [Padrão: ${CYAN}${CURRENT_DNS_DOMAIN}${NC}]:"
  read -rp "   Domínio DNS [${CURRENT_DNS_DOMAIN}]: " USER_DOMAIN < /dev/tty
  TARGET_DOMAIN="${USER_DOMAIN:-$CURRENT_DNS_DOMAIN}"

  echo -e "\n5. IP do Roteador Central (Core/Backbone) [Padrão: ${CYAN}${CURRENT_PING_CORE}${NC}]:"
  read -rp "   IP do Core [${CURRENT_PING_CORE}]: " USER_CORE < /dev/tty
  TARGET_CORE="${USER_CORE:-$CURRENT_PING_CORE}"

  echo -e "\n6. Senha do Administrador Web (admin) [Pressione ENTER para manter admin123]:"
  read -rp "   Nova Senha: " USER_PASS < /dev/tty
  TARGET_PASS="${USER_PASS:-admin123}"
else
  TARGET_PORT="$CURRENT_PORT"
  TARGET_URL="$SUGGESTED_PUBLIC_URL"
  TARGET_DNS="$CURRENT_DNS_SERVER"
  TARGET_DOMAIN="$CURRENT_DNS_DOMAIN"
  TARGET_CORE="$CURRENT_PING_CORE"
  TARGET_PASS="admin123"
fi

# Garante protocolo http/https na URL pública
if [[ ! "$TARGET_URL" =~ ^https?:// ]]; then
  TARGET_URL="http://${TARGET_URL}"
fi
TARGET_URL="${TARGET_URL%/}"

# Atualiza .env com as definições finais
sed -i "s|^PORT=.*|PORT=${TARGET_PORT}|g" .env 2>/dev/null || true
sed -i "s|^PUBLIC_SERVER_URL=.*|PUBLIC_SERVER_URL=${TARGET_URL}|g" .env 2>/dev/null || true
sed -i "s|^DNS_SERVER_TARGET=.*|DNS_SERVER_TARGET=${TARGET_DNS}|g" .env 2>/dev/null || true
sed -i "s|^DNS_DOMAIN_TARGET=.*|DNS_DOMAIN_TARGET=${TARGET_DOMAIN}|g" .env 2>/dev/null || true
sed -i "s|^PING_CORE_TARGET=.*|PING_CORE_TARGET=${TARGET_CORE}|g" .env 2>/dev/null || true
sed -i "s|^AUTH_ADMIN_PASSWORD=.*|AUTH_ADMIN_PASSWORD=${TARGET_PASS}|g" .env 2>/dev/null || true
sed -i "s|^SETUP_COMPLETED=.*|SETUP_COMPLETED=true|g" .env 2>/dev/null || true

# Cria marcador em disco indicando setup concluído
mkdir -p config
touch config/.setup_done 2>/dev/null || true

# Gera Session Secret seguro se ainda estiver o valor de exemplo
if grep -q "super-secret-session-key-change-in-prod-2026" .env 2>/dev/null; then
  NEW_SECRET=$(head /dev/urandom | tr -dc A-Za-z0-9 | head -c 48 2>/dev/null || echo "uepg-acon-telemetry-key-$(date +%s)")
  sed -i "s|SESSION_SECRET=super-secret-session-key-change-in-prod-2026|SESSION_SECRET=${NEW_SECRET}|g" .env 2>/dev/null || true
fi

echo -e "\n${GREEN}[✓] Configurações salvas no .env com sucesso!${NC}"
echo -e "    Porta: ${CYAN}${TARGET_PORT}${NC} | URL: ${CYAN}${TARGET_URL}${NC} | DNS: ${CYAN}${TARGET_DNS}${NC} (${CYAN}${TARGET_DOMAIN}${NC}) | Core: ${CYAN}${TARGET_CORE}${NC}\n"

# ==============================================================================
# 3. PREPARAÇÃO DOS DIRETÓRIOS E PERMISSÕES
# ==============================================================================
echo -e "${GREEN}[3/5] Preparando diretórios de armazenamento e downloads...${NC}"
mkdir -p mongo_data config/machines public/downloads
$SUDO chmod -R 775 mongo_data config 2>/dev/null || true

# Garante que os scripts em public/downloads sejam executáveis
chmod +x public/downloads/*.sh 2>/dev/null || true
chmod +x public/downloads/uepg-probe-monitor 2>/dev/null || true
echo -e "${GREEN}[✓] Diretórios e permissões organizados com sucesso.${NC}\n"

# ==============================================================================
# 4. EXECUÇÃO DO DEPLOY (DOCKER OU NATIVO)
# ==============================================================================
if [ "$MODE" = "docker" ]; then
  echo -e "${GREEN}[4/5] Construindo imagens e iniciando containers Docker...${NC}"
  
  # Exporta variáveis para o compose ler imediatamente
  export PORT="${TARGET_PORT}"
  export PUBLIC_SERVER_URL="${TARGET_URL}"

  # Para eventuais containers legados sem apagar volumes de banco
  $DOCKER_COMPOSE down --remove-orphans 2>/dev/null || true

  # Compila e sobe os containers em segundo plano
  $DOCKER_COMPOSE build --pull
  $DOCKER_COMPOSE up -d

  echo -e "${GREEN}[✓] Containers iniciados no Docker Compose!${NC}\n"
else
  echo -e "${GREEN}[4/5] Executando build e deploy nativo via Go + Systemd...${NC}"
  if ! command -v go >/dev/null 2>&1; then
    echo -e "${RED}[ERRO] Go 1.22+ não está instalado no sistema. Instale o Go ou use './deploy.sh' com Docker.${NC}"
    exit 1
  fi
  go build -ldflags="-w -s" -o central-server main.go
  $SUDO cp systemd/network-monitor.service /etc/systemd/system/ 2>/dev/null || true
  $SUDO systemctl daemon-reload
  $SUDO systemctl enable --now network-monitor
  $SUDO systemctl restart network-monitor
  echo -e "${GREEN}[✓] Serviço nativo reiniciado via systemd!${NC}\n"
fi

# ==============================================================================
# 5. VERIFICAÇÃO DE SAÚDE (HEALTHCHECK)
# ==============================================================================
echo -e "${GREEN}[5/5] Aguardando inicialização completa do serviço...${NC}"

HEALTH_URL="http://127.0.0.1:${TARGET_PORT}/health"
ATTEMPTS=0
MAX_ATTEMPTS=20
HEALTHY=false

while [ $ATTEMPTS -lt $MAX_ATTEMPTS ]; do
  if curl -sf "$HEALTH_URL" >/dev/null 2>&1; then
    HEALTHY=true
    break
  fi
  ATTEMPTS=$((ATTEMPTS + 1))
  sleep 1.5
  printf "."
done
printf "\n"

if [ "$HEALTHY" = true ]; then
  echo -e "\n${BLUE}================================================================================${NC}"
  echo -e "${GREEN}${BOLD}     ✓ DEPLOY CONCLUÍDO COM SUCESSO! SERVIDOR CENTRAL OPERACIONAL              ${NC}"
  echo -e "${BLUE}================================================================================${NC}"
  echo -e "  • ${BOLD}Dashboard Web:${NC}     ${CYAN}${TARGET_URL}${NC}"
  echo -e "  • ${BOLD}Acesso Local:${NC}      ${CYAN}http://${LOCAL_IP}:${TARGET_PORT}${NC}"
  echo -e "  • ${BOLD}Usuário Admin:${NC}     ${YELLOW}admin${NC} / Senha: ${YELLOW}admin123${NC}"
  echo -e "  • ${BOLD}Usuário Operador:${NC}  ${YELLOW}operador${NC} / Senha: ${YELLOW}operador123${NC}"
  echo -e "  • ${BOLD}Healthcheck:${NC}       ${GREEN}ONLINE (200 OK)${NC}\n"
  echo -e "${BLUE}Comandos úteis para o dia a dia:${NC}"
  if [ "$MODE" = "docker" ]; then
    echo -e "  • Ver logs ao vivo:    ${YELLOW}docker compose logs -f central-server${NC}"
    echo -e "  • Consultar status:    ${YELLOW}./deploy.sh --status${NC} ou ${YELLOW}docker compose ps${NC}"
    echo -e "  • Reiniciar ambiente:  ${YELLOW}docker compose restart${NC}"
    echo -e "  • Parar ambiente:      ${YELLOW}./deploy.sh --down${NC}\n"
  else
    echo -e "  • Ver logs ao vivo:    ${YELLOW}journalctl -u network-monitor -f${NC}"
    echo -e "  • Status do serviço:   ${YELLOW}systemctl status network-monitor${NC}\n"
  fi
  echo -e "${BLUE}================================================================================${NC}\n"
else
  echo -e "\n${YELLOW}[AVISO] O servidor está iniciando mas o healthcheck ainda não respondeu.${NC}"
  echo -e "Verifique os logs com: ${CYAN}docker compose logs central-server${NC}\n"
fi
