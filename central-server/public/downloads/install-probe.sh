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

# 10. Verificação e teste imediato
sleep 2
if systemctl is-active --quiet uepg-probe; then
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
else
  echo -e "${RED}[AVISO] O serviço foi criado mas parece não ter iniciado corretamente.${NC}"
  echo -e "Verifique o log com: ${YELLOW}journalctl -u uepg-probe -xe${NC}"
fi
