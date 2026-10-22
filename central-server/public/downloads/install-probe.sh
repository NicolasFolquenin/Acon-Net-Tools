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

# Determina valor padrão ou sugerido da localização
SUGGESTED_LOCATION="Campus Uvaranas"
if [ -f "/etc/uepg-probe/config.json" ]; then
  EXISTING_LOC=$(grep -o '"location":[[:space:]]*"[^"]*"' /etc/uepg-probe/config.json 2>/dev/null | cut -d'"' -f4 || true)
  if [ -n "$EXISTING_LOC" ] && [ "$EXISTING_LOC" != "Debian Linux Server" ]; then
    SUGGESTED_LOCATION="$EXISTING_LOC"
  fi
fi

LOCATION=""
if [ -c /dev/tty ]; then
  echo -e "Informe o Bloco / Localização física desta máquina (ex: Bloco L - NTI, Reitoria, etc.)"
  read -rp "Localização [${SUGGESTED_LOCATION}]: " USER_INPUT_LOC < /dev/tty
  LOCATION="${USER_INPUT_LOC:-$SUGGESTED_LOCATION}"
elif [ -t 0 ]; then
  echo -e "Informe o Bloco / Localização física desta máquina (ex: Bloco L - NTI, Reitoria, etc.)"
  read -rp "Localização [${SUGGESTED_LOCATION}]: " USER_INPUT_LOC
  LOCATION="${USER_INPUT_LOC:-$SUGGESTED_LOCATION}"
else
  LOCATION="$SUGGESTED_LOCATION"
fi
echo -e "${GREEN}[✓] Localização definida como: ${CYAN}${LOCATION}${NC}\n"

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

# 4. Criação dos diretórios de sistema e parada de serviço ativo para atualização
echo -e "${GREEN}[+] Criando diretórios em /etc/uepg-probe e /var/lib/uepg-probe...${NC}"
mkdir -p /etc/uepg-probe
mkdir -p /var/lib/uepg-probe
mkdir -p /usr/local/bin

if systemctl is-active --quiet uepg-probe 2>/dev/null; then
  echo -e "${YELLOW}[!] Parando serviço anterior para atualização do executável...${NC}"
  systemctl stop uepg-probe 2>/dev/null || true
fi

# 5. Download do binário (com arquivo temporário para evitar Text File Busy)
DOWNLOAD_URL="${SERVER_URL}/downloads/${BIN_NAME}"
TARGET_BIN="/usr/local/bin/uepg-probe"
TMP_BIN="/usr/local/bin/uepg-probe.tmp.$$"

echo -e "${GREEN}[+] Baixando executável de: ${CYAN}${DOWNLOAD_URL}${NC}..."
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$DOWNLOAD_URL" -o "$TMP_BIN"
elif command -v wget >/dev/null 2>&1; then
  wget -q "$DOWNLOAD_URL" -O "$TMP_BIN"
else
  echo -e "${RED}[ERRO] Nem 'curl' nem 'wget' foram encontrados no sistema.${NC}"
  echo -e "Instale um deles com: ${YELLOW}apt update && apt install -y curl${NC}"
  exit 1
fi

chmod +x "$TMP_BIN"
mv -f "$TMP_BIN" "$TARGET_BIN"
echo -e "${GREEN}[+] Binário instalado com sucesso em ${CYAN}${TARGET_BIN}${NC}"

# 6. Criação ou atualização do arquivo de configuração /etc/uepg-probe/config.json
CONFIG_FILE="/etc/uepg-probe/config.json"
HOSTNAME=$(cat /etc/hostname 2>/dev/null || uname -n)

if [ -f "$CONFIG_FILE" ]; then
  echo -e "${GREEN}[+] Atualizando configurações no arquivo ${CYAN}${CONFIG_FILE}${NC}..."
  if command -v sed >/dev/null 2>&1; then
    sed -i "s|\"server_url\":[[:space:]]*\"[^\"]*\"|\"server_url\": \"${SERVER_URL}\"|g" "$CONFIG_FILE"
    sed -i "s|\"location\":[[:space:]]*\"[^\"]*\"|\"location\": \"${LOCATION}\"|g" "$CONFIG_FILE"
    # Remove node_id do JSON para garantir que o Hostname seja sempre obtido diretamente do SO
    sed -i '/"node_id":/d' "$CONFIG_FILE"
  fi
else
  echo -e "${GREEN}[+] Gerando arquivo de configuração padrão em ${CYAN}${CONFIG_FILE}${NC}..."
  cat <<EOF > "$CONFIG_FILE"
{
  "location": "${LOCATION}",
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

# Animação Matemática de Ondas WiFi em ASCII (estilo Donut.c)
show_wifi_animation() {
  local srv="$1"
  local host="$2"
  local ip="$3"
  local gw="$4"

  trap '
    printf "\033[?25h\n\n"
    echo -e "${GREEN}==================================================================${NC}"
    echo -e "${GREEN}  ✓ Sonda mantida em SEGUNDO PLANO (serviço ativo no systemd)!   ${NC}"
    echo -e "${GREEN}==================================================================${NC}"
    echo -e "A sonda UEPG Net Tools permanece ativa e enviando dados para o servidor."
    echo -e "Comandos rápidos úteis:"
    echo -e "  • Logs em tempo real: ${YELLOW}journalctl -u uepg-probe -f${NC}"
    echo -e "  • Consultar status:   ${YELLOW}systemctl status uepg-probe${NC}"
    echo -e "  • Reiniciar sonda:    ${YELLOW}systemctl restart uepg-probe${NC}\n"
    exit 0
  ' SIGINT SIGTERM

  printf "\033[?25l"
  echo -e "\n${BLUE}================================================================================${NC}"
  echo -e "${CYAN}   📶 UEPG NET TOOLS • TRANSMISSÃO WIFI EM TEMPO REAL (MATH ASCII WAVES)         ${NC}"
  echo -e "${BLUE}================================================================================${NC}\n"

  F_0=(
    '                                    '
    '                                    '
    '                                    '
    '           .,,,-------,,,.          '
    '        .,,,,,..     ..,,,,,.       '
    '      .,,,.               .,,,.     '
    '    ..,.                     .,..   '
    '   ...         ,-~~~-,         ...  '
    '  ...       ,:;=======;:,       ... '
    '  .       ,~::~,     ,~::~,       . '
    '         .-~-           -~-.        '
    '         .,.     /|\\     .,.        '
    '               ( ~@~ )              '
    '              [=======]             '
  )
  F_1=(
    '                                    '
    '                                    '
    '               .......              '
    '          .,,,,,,,,,,,,,,,.         '
    '       .,,,,..         ..,,,,.      '
    '     ..,..                 ..,..    '
    '    ...                       ...   '
    '   ..        .-~:;;;:~-.        ..  '
    '  ..       ,~;;;;;:;;;;;~,       .. '
    '         .-~:~,       ,~:~-.        '
    '         ,--.           .--,        '
    '         ..      /|\\      ..        '
    '               (.;o;.)              '
    '              [=======]             '
  )
  F_2=(
    '                                    '
    '                                    '
    '             ....,,,....            '
    '         ..,,,,,,,.,,,,,,,..        '
    '      ..,,..             ..,,..     '
    '     ....                   ....    '
    '   ...                         ...  '
    '  ...       .-~;;===;;~-.       ... '
    '  .       .-:;;:~---~:;;:-.       . '
    '         ,~~~,         ,~~~,        '
    '        .,-,             ,-,.       '
    '         ..      /|\\      ..        '
    '               (-=@=-)              '
    '              [=======]             '
  )
  F_3=(
    '                                    '
    '                                    '
    '            ...,,,,,,,...           '
    '        ..,,,....   ....,,,..       '
    '      ....                 ....     '
    '    ....                     ....   '
    '   ..          .,,-,,.          ..  '
    '  ..       .-~;;=====;;~-.       .. '
    '         .-~::~,.   .,~::~-.        '
    '        .-~-,           ,-~-.       '
    '        ,,,               ,,,       '
    '         .      ,/|\\,      .        '
    '               (~=o=~)              '
    '              [=======]             '
  )
  F_4=(
    '                                    '
    '                                    '
    '           .....,,,,,.....          '
    '        ......         ......       '
    '     ....                   ....    '
    '    ..                         ..   '
    '   .          ,-~~:~~-,          .  '
    '           ,~:;;;:::;;;:~,          '
    '         ,~:~-.       .-~:~,        '
    '        ,--,             ,--,       '
    '       .,,.               .,,.      '
    '               .:/|\\:.              '
    '               (~;@;~)              '
    '              [=======]             '
  )
  F_5=(
    '                                    '
    '                  .                 '
    '          .................         '
    '       .....             .....      '
    '     ...                     ...    '
    '    .                           .   '
    '            .,~::;;;::~,.           '
    '          ,~::::~---~::::~,         '
    '        .-~~-.         .-~~-.       '
    '       .--,               ,--.      '
    '      .,,.       .,.       .,,.     '
    '               -=/|\\=-              '
    '               (~~o~~)              '
    '              [=======]             '
  )
  F_6=(
    '                                    '
    '                 ...                '
    '          ......     ......         '
    '       ...                 ...      '
    '     ..                       ..    '
    '                ..,..               '
    '           .-~::;;;;;::~-.          '
    '         .-~:~~,.. ..,~~:~-.        '
    '        ,-~-.           .-~-,       '
    '       ,-,.               .,-,      '
    '      .,.       -:;:-       .,.     '
    '              ,:!/|\\!:,             '
    '               (-.@.-)              '
    '              [=======]             '
  )
  F_7=(
    '                                    '
    '                 ...                '
    '          ..             ..         '
    '                                    '
    '                                    '
    '              .,-----,.             '
    '          .,~:::::::::::~,.         '
    '        .-~~~-.       .-~~~-.       '
    '       .---.             .---.      '
    '      .,,.                 .,,.     '
    '      ...      -;!*!;-      ...     '
    '              ~;!/|\\!;~             '
    '               (. o .)              '
    '              [=======]             '
  )
  F_8=(
    '                                    '
    '                                    '
    '                                    '
    '                                    '
    '                                    '
    '            .,-~~~~~~~-,.           '
    '         .,-~::~~---~~::~-,.        '
    '        ,-~-,.         .,-~-,       '
    '      .,--.               .--,.     '
    '     .,,.         .         .,,.    '
    '     ...      ,:!*#*!:,      ...    '
    '             ,:;;/|\\;;:,            '
    '               (  @  )              '
    '              [=======]             '
  )
  F_9=(
    '                                    '
    '                                    '
    '                                    '
    '                                    '
    '                  .                 '
    '           .,-~~:::::~~-,.          '
    '         ,-~~~-,,...,,-~~~-,        '
    '       .---,             ,---.      '
    '      ,,-.                 .-,,     '
    '     .,,        ,~~~,        ,,.    '
    '     ..      .~=**#**=~.      ..    '
    '            .-::-/|\\-::-.           '
    '               (  o  )              '
    '              [=======]             '
  )
  F_10=(
    '                                    '
    '                                    '
    '                                    '
    '                                    '
    '              ..,,,,,..             '
    '          .,-~~~~~~~~~~~-,.         '
    '        .-~~-,.       .,-~~-.       '
    '      .,--,               ,--,.     '
    '     .,,,                   ,,,.    '
    '    ..,.       -:;=;:-       .,..   '
    '    ...      -;!!!!!!!;-      ...   '
    '            ,~~- /|\\ -~~,           '
    '               (  @  )              '
    '              [=======]             '
  )
  F_11=(
    '                                    '
    '                                    '
    '                                    '
    '                                    '
    '             .,,-----,,.            '
    '         .,-~~~~~---~~~~~-,.        '
    '       .,---,.         .,---,.      '
    '      ,,-,                 ,-,,     '
    '     .,,                     ,,.    '
    '    ...       -:=!*!=:-       ...   '
    '    ..      ,:;=;:::;=;:,      ..   '
    '           .-~-. /|\\ .-~-.          '
    '               (  o  )              '
    '              [=======]             '
  )
  F_12=(
    '                                    '
    '                                    '
    '                                    '
    '                                    '
    '           ..,--~~~~~--,..          '
    '        .,--~--,,,,,,,--~--,.       '
    '      .,---.             .---,.     '
    '     .,,,.                 .,,,.    '
    '    .,,.        .,-,.        .,,.   '
    '    ..       ,:=!***!=:,       ..   '
    '   ..      .-:;:-,.,-:;:-.      ..  '
    '           ,--.  /|\\  .--,          '
    '               (  @  )              '
    '              [=======]             '
  )
  F_13=(
    '                                    '
    '                                    '
    '                                    '
    '               .......              '
    '          .,,--~~~~~~~--,,.         '
    '        .,---,..     ..,---,.       '
    '      .,-,.               .,-,.     '
    '     .,,.                   .,,.    '
    '    .,.        ,~:::~,        .,.   '
    '   ...      ,~;!!!!!!!;~,      ...  '
    '   .       ,~::-     -::~,       .  '
    '          .,-,   /|\\   ,-,.         '
    '               (  o  )              '
    '              [=======]             '
  )
  F_14=(
    '                                    '
    '                                    '
    '                                    '
    '             ..,,,,,,,..            '
    '         .,,-------------,,.        '
    '       .,--,,.         .,,--,.      '
    '     .,,,.                 .,,,.    '
    '    .,,.                     .,,.   '
    '   ...       .-:;===;:-.       ...  '
    '   ..      .-;==;;:;;==;-.      ..  '
    '          .-:~,       ,~:-.         '
    '          .,,    /|\\    ,,.         '
    '               (  @  )              '
    '              [=======]             '
  )
  F_15=(
    '                                    '
    '                                    '
    '                                    '
    '            ..,,,---,,,..           '
    '        ..,---,,,,,,,,,---,..       '
    '      .,,,,.             .,,,,.     '
    '     .,,.                   .,,.    '
    '   ..,.          ...          .,..  '
    '   ..        -:;=!!!=;:-        ..  '
    '  ..       ,:;;:~,,,~:;;:,       .. '
    '          ,~~-         -~~,         '
    '         .,,.    /|\\    .,,.        '
    '               ( .o. )              '
    '              [=======]             '
  )

  local sec=0
  local frame_idx=0
  local num_frames=16
  local signals=("[ ▂   ]" "[ ▂▄  ]" "[ ▂▄▆ ]" "[ ▂▄▆█]")

  while true; do
    local sig="${signals[$(( (frame_idx / 2) % 4 ))]}"
    local mm=$(( sec / 60 ))
    local ss=$(( sec % 60 ))
    local time_str=$(printf "%02d:%02d" $mm $ss)

    eval "local -a cur_f=(\"\${F_${frame_idx}[@]}\")"

    printf "\r\033[K ${CYAN}%-36s${NC}   ${GREEN}● STATUS: ONLINE${NC}  ${BLUE}[ Systemd Daemon ]${NC}\n" "${cur_f[0]}"
    printf "\033[K ${CYAN}%-36s${NC}   ${BLUE}──────────────────────────────────────────────${NC}\n" "${cur_f[1]}"
    printf "\033[K ${CYAN}%-36s${NC}   • Servidor:   ${CYAN}%s${NC}\n" "${cur_f[2]}" "${srv}"
    printf "\033[K ${CYAN}%-36s${NC}   • Hostname:   ${YELLOW}%s${NC}\n" "${cur_f[3]}" "${host}"
    printf "\033[K ${CYAN}%-36s${NC}   • IP Sonda:   ${CYAN}%s${NC}\n" "${cur_f[4]}" "${ip}"
    printf "\033[K ${CYAN}%-36s${NC}   • Gateway:    ${CYAN}%s${NC}\n" "${cur_f[5]}" "${gw}"
    printf "\033[K ${CYAN}%-36s${NC}   • Conexão:    ${GREEN}%s${NC}  • Uptime: ${YELLOW}%s${NC}\n" "${cur_f[6]}" "${sig}" "${time_str}"
    printf "\033[K ${CYAN}%-36s${NC}   • Telemetria: Coleta em tempo real ativa\n" "${cur_f[7]}"
    printf "\033[K ${CYAN}%-36s${NC}   ${BLUE}──────────────────────────────────────────────${NC}\n" "${cur_f[8]}"
    printf "\033[K ${CYAN}%-36s${NC}   ${YELLOW}👉 Pressione [ CTRL + C ] para sair${NC}\n" "${cur_f[9]}"
    printf "\033[K ${CYAN}%-36s${NC}      ${GREEN}(a sonda continuará em segundo plano)${NC}\n" "${cur_f[10]}"
    printf "\033[K ${CYAN}%-36s${NC}\n" "${cur_f[11]}"
    printf "\033[K ${CYAN}%-36s${NC}\n" "${cur_f[12]}"
    printf "\033[K ${CYAN}%-36s${NC}\n" "${cur_f[13]}"

    sleep 0.15
    if [ $(( frame_idx % 6 )) -eq 0 ]; then
      sec=$(( sec + 1 ))
    fi
    frame_idx=$(( (frame_idx + 1) % num_frames ))

    # Retorna 14 linhas para cima para redesenhar fluidamente o próximo frame das ondas WiFi
    printf "\033[14A"
  done
}



# 10. Verificação e teste imediato
sleep 2
if systemctl is-active --quiet uepg-probe; then
  # Detecção robusta do IP local da máquina
  LOCAL_IP=$(ip -4 route get 1.1.1.1 2>/dev/null | awk '{print $7}' | tr -d '\n')
  if [ -z "$LOCAL_IP" ]; then
    LOCAL_IP=$(ip -4 addr show up scope global 2>/dev/null | grep -oP '(?<=inet\s)\d+(\.\d+){3}' | head -n1)
  fi
  if [ -z "$LOCAL_IP" ]; then
    LOCAL_IP=$(hostname -I 2>/dev/null | awk '{print $1}')
  fi
  [ -z "$LOCAL_IP" ] && LOCAL_IP="127.0.0.1"

  # Detecção robusta do Gateway padrão
  GATEWAY_IP=$(ip route show default 2>/dev/null | awk '/default/ {print $3}' | head -n1)
  if [ -z "$GATEWAY_IP" ]; then
    GATEWAY_IP=$(ip route 2>/dev/null | awk '/default/ {print $3}' | head -n1)
  fi
  [ -z "$GATEWAY_IP" ] && GATEWAY_IP="Automático"

  # Se houver terminal disponível, inicia a animação de ondas WiFi em tempo real (ASCII math)
  if [ -t 1 ] || [ -c /dev/tty ]; then
    show_wifi_animation "${SERVER_URL}" "${HOSTNAME}" "${LOCAL_IP}" "${GATEWAY_IP}"
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
