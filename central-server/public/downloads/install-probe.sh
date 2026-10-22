#!/usr/bin/env bash
# ==============================================================================
# Script de Instalação Automatizada da Sonda UEPG Net Tools para Debian / Linux
# ==============================================================================
set -e

# Cores para terminal
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${BLUE}==================================================================${NC}"
echo -e "${CYAN}   UEPG Net Tools - Instalador da Sonda Client (Linux / Debian)   ${NC}"
echo -e "${BLUE}==================================================================${NC}"

# 1. Verifica se está rodando como root
if [ "$EUID" -ne 0 ]; then
  echo -e "${RED}[ERRO] Este instalador precisa de privilégios de superusuário (root).${NC}"
  echo -e "Por favor, execute novamente usando: ${YELLOW}sudo bash $0${NC} ou ${YELLOW}curl ... | sudo bash${NC}"
  exit 1
fi

# 2. Determina a URL do Servidor Central (Interativo via /dev/tty ou parâmetros)
SUGGESTED_URL="${SERVER_URL:-$1}"
if [ -z "$SUGGESTED_URL" ] && [ -n "$PROBE_SERVER" ]; then
  SUGGESTED_URL="$PROBE_SERVER"
fi
if [ -z "$SUGGESTED_URL" ] && [ -f "/etc/uepg-probe/config.json" ]; then
  SUGGESTED_URL=$(grep -o '"server_url":[[:space:]]*"[^"]*"' /etc/uepg-probe/config.json 2>/dev/null | cut -d'"' -f4 || true)
fi
if [ -z "$SUGGESTED_URL" ]; then
  SUGGESTED_URL="https://akvorado.acon.dev.uepg.br"
fi

echo -e "\n${BLUE}------------------------------------------------------------------${NC}"
echo -e "${YELLOW}▶ Configuração de Conexão com o Servidor Central${NC}"
echo -e "${BLUE}------------------------------------------------------------------${NC}"

SERVER_URL=""

# Lê diretamente do terminal controlador (/dev/tty), mesmo quando executado via curl piping
if [ -c /dev/tty ]; then
  echo -e "Informe a URL do Servidor Central onde as medições serão salvas."
  echo -e "Pressione ${GREEN}[ENTER]${NC} para aceitar o valor sugerido: ${CYAN}${SUGGESTED_URL}${NC}"
  echo ""
  read -rp "URL do Servidor Central [${SUGGESTED_URL}]: " USER_INPUT_URL < /dev/tty
  if [ -n "$USER_INPUT_URL" ]; then
    SERVER_URL="$USER_INPUT_URL"
  else
    SERVER_URL="$SUGGESTED_URL"
  fi
elif [ -t 0 ]; then
  echo -e "Informe a URL do Servidor Central onde as medições serão salvas."
  echo -e "Pressione ${GREEN}[ENTER]${NC} para aceitar o valor sugerido: ${CYAN}${SUGGESTED_URL}${NC}"
  echo ""
  read -rp "URL do Servidor Central [${SUGGESTED_URL}]: " USER_INPUT_URL
  if [ -n "$USER_INPUT_URL" ]; then
    SERVER_URL="$USER_INPUT_URL"
  else
    SERVER_URL="$SUGGESTED_URL"
  fi
else
  # Modo não-interativo / CI
  SERVER_URL="$SUGGESTED_URL"
fi

# Ajuste automático se o usuário digitou sem http:// ou https://
if [[ ! "$SERVER_URL" =~ ^https?:// ]]; then
  if [[ "$SERVER_URL" =~ ^(localhost|127\.|192\.168\.|10\.|172\.) ]]; then
    SERVER_URL="http://${SERVER_URL}"
  else
    SERVER_URL="https://${SERVER_URL}"
  fi
fi

SERVER_URL="${SERVER_URL%/}"
echo -e "${GREEN}[✓] Servidor Central configurado como: ${CYAN}${SERVER_URL}${NC}\n"

# 3. Detecta arquitetura do sistema
ARCH=$(uname -m)
BIN_NAME=""

case "$ARCH" in
  x86_64|amd64)
    BIN_NAME="uepg-probe-linux-amd64"
    ;;
  aarch64|arm64)
    BIN_NAME="uepg-probe-linux-arm64"
    ;;
  *)
    echo -e "${RED}[ERRO] Arquitetura de processador '$ARCH' não é suportada diretamente pelos binários pré-compilados.${NC}"
    exit 1
    ;;
esac

echo -e "${GREEN}[+] Arquitetura detectada: ${CYAN}${ARCH}${GREEN} -> Binário: ${CYAN}${BIN_NAME}${NC}"

# 4. Criação dos diretórios de sistema
echo -e "${GREEN}[+] Criando diretórios em /etc/uepg-probe e /var/lib/uepg-probe...${NC}"
mkdir -p /etc/uepg-probe
mkdir -p /var/lib/uepg-probe

# 5. Download do binário
DOWNLOAD_URL="${SERVER_URL}/downloads/${BIN_NAME}"
TARGET_BIN="/usr/local/bin/uepg-probe"

echo -e "${GREEN}[+] Baixando executável de: ${CYAN}${DOWNLOAD_URL}${NC}..."
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$DOWNLOAD_URL" -o "$TARGET_BIN"
elif command -v wget >/dev/null 2>&1; then
  wget -q "$DOWNLOAD_URL" -O "$TARGET_BIN"
else
  echo -e "${RED}[ERRO] Nem 'curl' nem 'wget' foram encontrados no sistema.${NC}"
  echo -e "Instale um deles com: ${YELLOW}apt update && apt install -y curl${NC}"
  exit 1
fi

chmod +x "$TARGET_BIN"
echo -e "${GREEN}[+] Binário instalado com sucesso em ${CYAN}${TARGET_BIN}${NC}"

# 6. Criação ou atualização do arquivo de configuração /etc/uepg-probe/config.json
CONFIG_FILE="/etc/uepg-probe/config.json"
HOSTNAME=$(cat /etc/hostname 2>/dev/null || uname -n)

if [ -f "$CONFIG_FILE" ]; then
  echo -e "${GREEN}[+] Atualizando 'server_url' para ${CYAN}${SERVER_URL}${GREEN} no arquivo ${CYAN}${CONFIG_FILE}${NC}..."
  if command -v sed >/dev/null 2>&1; then
    sed -i "s|\"server_url\":[[:space:]]*\"[^\"]*\"|\"server_url\": \"${SERVER_URL}\"|g" "$CONFIG_FILE"
  fi
else
  echo -e "${GREEN}[+] Gerando arquivo de configuração padrão em ${CYAN}${CONFIG_FILE}${NC}..."
  cat <<EOF > "$CONFIG_FILE"
{
  "node_id": "${HOSTNAME}",
  "location": "Debian Linux Server",
  "server_url": "${SERVER_URL}",
  "interval_minutes": 60,
  "ping_target": "auto_gateway",
  "sync_with_central": true
}
EOF
fi

# 7. Sugestão de pacote lldpd para captura de porta de switch
if ! command -v lldpcli >/dev/null 2>&1; then
  echo -e "${YELLOW}[DICA] Para habilitar a identificação automática de Switch/Porta via LLDP no Debian:${NC}"
  echo -e "       ${CYAN}apt update && apt install -y lldpd && systemctl enable --now lldpd${NC}"
fi

# 8. Criação da unidade de serviço systemd
SERVICE_FILE="/etc/systemd/system/uepg-probe.service"
echo -e "${GREEN}[+] Configurando serviço systemd em ${CYAN}${SERVICE_FILE}${NC}..."

cat <<EOF > "$SERVICE_FILE"
[Unit]
Description=UEPG Net Tools - Sonda de Telemetria de Rede (Client Probe)
Documentation=https://github.com/uepg/net-tools
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=/var/lib/uepg-probe
ExecStart=/usr/local/bin/uepg-probe -config /etc/uepg-probe/config.json
Restart=always
RestartSec=15
StandardOutput=journal
StandardError=journal
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

# 9. Recarrega e ativa o serviço
echo -e "${GREEN}[+] Ativando e iniciando o serviço uepg-probe no systemd...${NC}"
systemctl daemon-reload
systemctl enable uepg-probe
systemctl restart uepg-probe

# Função de animação de telemetria de rede e tráfego
show_network_animation() {
  local srv="$1"
  local host="$2"
  local ip="$3"
  local gw="$4"

  # Trap para interceptar o Ctrl+C e confirmar a execução em segundo plano
  trap '
    printf "\033[?25h\n\n"
    echo -e "${GREEN}==================================================================${NC}"
    echo -e "${GREEN}  ✓ Sonda operando com sucesso em SEGUNDO PLANO (systemd)!       ${NC}"
    echo -e "${GREEN}==================================================================${NC}"
    echo -e "A sonda UEPG Net Tools permanece ativa e enviando dados para o servidor."
    echo -e "Comandos rápidos úteis:"
    echo -e "  • Logs em tempo real: ${YELLOW}journalctl -u uepg-probe -f${NC}"
    echo -e "  • Consultar status:   ${YELLOW}systemctl status uepg-probe${NC}"
    echo -e "  • Reiniciar sonda:    ${YELLOW}systemctl restart uepg-probe${NC}\n"
    exit 0
  ' SIGINT SIGTERM

  # Esconde o cursor
  printf "\033[?25l"

  local frames=(
    "───( ●  ○  ○ )──[ 🌐 INTERNET ]──────────────>"
    "────( ○  ●  ○ )─[ 🌐 INTERNET ]──────────────>"
    "─────( ○  ○  ● )[ 🌐 INTERNET ]──────────────>"
    "────────────────[ 🌐 INTERNET ]──( ●  ○  ○ )─>"
    "────────────────[ 🌐 INTERNET ]───( ○  ●  ○ )─>"
    "<───( ●  ○  ○ )─[ 🌐 INTERNET ]────────────────"
    "<────( ○  ●  ○ )[ 🌐 INTERNET ]────────────────"
    "<─────( ○  ○  ●)[ 🌐 INTERNET ]────────────────"
  )
  local globes=("🌍" "🌎" "🌏")
  local signals=("[ ▂   ]" "[ ▂▄  ]" "[ ▂▄▆ ]" "[ ▂▄▆█]")

  local sec=0
  local f=0
  local total_frames=${#frames[@]}

  echo -e "\n${BLUE}==================================================================${NC}"
  echo -e "${CYAN}     🚀 SONDA ATIVA • MONITORAMENTO DE REDE EM TEMPO REAL         ${NC}"
  echo -e "${BLUE}==================================================================${NC}\n"

  while true; do
    local wire="${frames[$f]}"
    local globe="${globes[$(( (f / 2) % 3 ))]}"
    local sig="${signals[$(( f % 4 ))]}"
    local mm=$(( sec / 60 ))
    local ss=$(( sec % 60 ))
    local time_str=$(printf "%02d:%02d" $mm $ss)

    printf "\r\033[K ${CYAN}💻 [${host}]${NC} ${YELLOW}${wire}${NC} ${GREEN}☁ [Servidor Central]${NC}\n"
    printf "\033[K   ${BLUE}├─${NC} Status:     ${GREEN}● ONLINE${NC} (Daemon Systemd Ativo)\n"
    printf "\033[K   ${BLUE}├─${NC} Servidor:   ${CYAN}%s${NC}\n" "${srv}"
    printf "\033[K   ${BLUE}├─${NC} IP Local:   ${CYAN}%s${NC} | Gateway: ${CYAN}%s${NC}\n" "${ip}" "${gw}"
    printf "\033[K   ${BLUE}├─${NC} Conexão:    ${GREEN}%s${NC} %s Uptime: ${YELLOW}%s${NC}\n" "${sig}" "${globe}" "${time_str}"
    printf "\033[K   ${BLUE}└─${NC} ${YELLOW}👉 Pressione [ CTRL + C ] para voltar ao terminal e deixar em 2º plano${NC}\n"

    sleep 0.4
    sec=$(( sec + 1 ))
    f=$(( (f + 1) % total_frames ))

    # Retorna 6 linhas para cima para redesenhar fluidamente o próximo frame
    printf "\033[6A"
  done
}

# 10. Verificação e teste imediato
sleep 2
if systemctl is-active --quiet uepg-probe; then
  LOCAL_IP=$(hostname -I 2>/dev/null | awk '{print $1}')
  [ -z "$LOCAL_IP" ] && LOCAL_IP="127.0.0.1"
  GATEWAY_IP=$(ip route 2>/dev/null | awk '/default/ {print $3}' | head -n1)
  [ -z "$GATEWAY_IP" ] && GATEWAY_IP="Automático"

  # Se houver terminal disponível, inicia a animação de tráfego interativo
  if [ -t 1 ] || [ -c /dev/tty ]; then
    show_network_animation "${SERVER_URL}" "${HOSTNAME}" "${LOCAL_IP}" "${GATEWAY_IP}"
  else
    echo -e "\n${GREEN}==================================================================${NC}"
    echo -e "${GREEN}  ✓ Sonda instalada e rodando com sucesso no seu Debian/Linux!   ${NC}"
    echo -e "${GREEN}==================================================================${NC}"
    echo -e "A máquina já realizou o auto-registro e está visível no Dashboard central.\n"
    echo -e "Comandos úteis para gerenciar a sonda:"
    echo -e "  • Ver status:       ${YELLOW}systemctl status uepg-probe${NC}"
    echo -e "  • Ver logs ao vivo: ${YELLOW}journalctl -u uepg-probe -f${NC}"
    echo -e "  • Reiniciar:        ${YELLOW}systemctl restart uepg-probe${NC}"
    echo -e "  • Teste manual CLI: ${YELLOW}/usr/local/bin/uepg-probe -once${NC}"
    echo -e "  • Configuração:     ${CYAN}/etc/uepg-probe/config.json${NC}\n"
  fi
else
  echo -e "${RED}[AVISO] O serviço foi criado mas parece não ter iniciado corretamente.${NC}"
  echo -e "Verifique o log com: ${YELLOW}journalctl -u uepg-probe -xe${NC}"
fi

