# UEPG Net Tools — Como Utilizar o Net-Tools

> **Guia Oficial de Instalação, Operação e Uso Diário**  
> **Projeto:** Sistema Distribuído de Telemetria e Diagnóstico de Rede  
> **Instituição:** Universidade Estadual de Ponta Grossa (UEPG) — Anel de Conectividade (ACON) / NTI  
> **Versão:** v2.4.1  

---

## 1. Operando o Dashboard Central (Painel Web NOC)

O Dashboard Central é a interface gráfica onde administradores de rede, analistas de suporte e gestores acompanham o status de toda a rede universitária.

### 1.1. Acesso ao Painel
- **URL Padrão (Produção ACON):** `https://nettools.acon.uepg.br` *(ou diretamente via IP da VM: `http://177.101.22.40:8085`)*.
- **Autenticação:** O sistema possui suporte a autenticação de administrador com chave de sessão JWT/Cookie para operações sensíveis (como personalização de módulos de teste, reconfiguração de grupos e disparo de comandos).

---

### 1.2. Visão Global do Dashboard (`#dashboard`)
Ao abrir a aplicação, a tela principal exibe:

1. **Cabeçalho de KPIs Globais:**
   - **Total de Sondas:** Quantidade de computadores monitorados cadastrados.
   - **Sondas Online:** Quantidade de nós que enviaram telemetria nos últimos minutos (com indicação visual verde).
   - **Latência Média Geral:** Média aritmética dos pings de todos os nós em tempo real.
   - **Throughput Agregado:** Volume de tráfego de download e upload registrado pelas medições recentes.

2. **Gráficos Temporais Interativos (Chart.js):**
   - **Gráfico de Latência de Rede:** Exibe as curvas de tempo de resposta (ms). Ao selecionar uma máquina específica, o gráfico foca apenas no histórico daquela sonda.
   - **Gráfico de Perda de Pacotes:** Linha do tempo de integridade do enlace (0% a 100%).
   - **Gráfico de Throughput:** Taxas de transferência ao longo das horas do dia.

3. **Filtros Rápidos:**
   - Permite filtrar a listagem de máquinas por **Grupo** (ex: *Laboratórios de Informática*, *Reitoria*, *Biblioteca*), por **VLAN** (ex: *VLAN 1101*, *VLAN 102*) ou por **Localização Física**.

---

### 1.3. Painel Técnico de Máquina Única (Visualização Isolada de Sonda)
Ao clicar sobre qualquer máquina na tabela ou selecioná-la no menu suspenso de gráficos, abre-se o **Painel Técnico Dedicado**:

```
+---------------------------------------------------------------------------------------------------------+
| [ Ícone PC ]  VISUALIZAÇÃO ISOLADA DE SONDA: DESKTOP-H7U5DKH    [Online] [v2.4.1]   [ Forçar Ping ]     |
| IP: 10.0.101.52 | Grupo: ACON | VLAN: 1101 | Switch: ACONS0001 | Porta: GE1/0/4 | Usuário: João Ramos   |
+---------------------------------------------------------------------------------------------------------+
|                                                                                                         |
|  [ Card 1: Status Wi-Fi & Sinal ]   [ Card 2: Módulos Carregados ]    [ Card 3: Topologia Switch LLDP ] |
|  - Conexão: Cabo Ethernet           - Perfil: Padrão do Grupo (7)     - Nome: ACONS0001                 |
|  - Adaptador Wi-Fi: Desconectado    - Ping Gateway Local (Ativo)      - Porta: GE1/0/4                  |
|  - Sinal: -- (Inativo)              - Throughput ACON (Ativo)         - VLAN: 1101                      |
|                                     - [ Botão: Editar Módulos ]       - [ Botão: Ver Vizinhos Switch ]  |
|                                                                                                         |
|  [ Card 4: Inventário Completo de Placas de Rede & Endereços MAC ]                                      |
|  Status  | Interface / Adaptador     | Tipo      | MAC Físico          | IPv4         | Prioridade      |
|  Ativa   | Ethernet Realtek PCIe GbE | ETHERNET  | D0-C1-B5-81-F7-61   | 10.0.101.52  | Principal (GW)  |
|  Inativa | Intel Wi-Fi 7 BE200       | WI-FI     | E0-3A-AA-A7-A2-8D   | --           | Secundária      |
+---------------------------------------------------------------------------------------------------------+
```

#### Funcionalidades deste Painel:
- **Diagnóstico de Ping ICMP Forçado (`Forçar Ping`):**
  Dispara 4 pacotes ICMP em tempo real a partir do Servidor Central contra o IP da máquina. Exibe o terminal do comando, a latência média, mínima e máxima e a taxa de perda. O resultado é **gravado automaticamente no banco de dados**, atualizando os gráficos e a tabela sem necessidade de recarregar a página (F5).
- **Ver Vizinhos Conectados neste Switch:**
  Abre um modal exibindo todas as outras sondas de computadores que estão fisicamente conectadas no mesmo switch (ex: `ACONS0001`), facilitando o diagnóstico de quedas coletivas em um mesmo armário de telecomunicações.
- **Personalização de Módulos da Sonda (`Editar Módulos`):**
  Permite ativar ou desativar módulos específicos para aquela máquina (ex: desativar teste de throughput se o usuário estiver em reunião importante), ou clicar em *"Restaurar Grupo"* para voltar a herdar o perfil comum.

---

### 1.4. Gestão de Usuários do Sistema (`#users`)
Acessando a aba **Usuários** no menu lateral esquerdo:
- É possível inspecionar o inventário de usuários locais de qualquer máquina cadastrada.
- Exibe o usuário ativo no momento do teste (destacado em verde com o badge `Logado`).
- Exibe a lista completa de contas do Windows/Linux (Administrador, Usuário Padrão, Contas do Sistema), mesmo se a máquina estiver desligada no momento.

---

## 2. Utilizando e Instalando a Sonda Cliente (`client-probe`)

### 2.1. No Windows (Desktop com Interface Gráfica Wails)

#### A. Modo de Teste Rápido no Ambiente de Desenvolvimento
Dentro da pasta `client-probe`, existe um utilitário de inicialização rápida em lote:
```cmd
cd client-probe
testar.bat
```
O script `testar.bat` automaticamente:
1. Encerra instâncias anteriores do aplicativo que possam estar presas em memória.
2. Libera silenciosamente a regra de firewall do Windows para permitir respostas de ping ICMP (`FPS-ICMP4-ERQ-In`).
3. Sincroniza os arquivos do frontend HTML/CSS/JS.
4. Compila o binário com suporte a janelas Wails (`go build -tags desktop,production`).
5. Abre a janela gráfica da sonda na tela exibindo o status em tempo real.

#### B. Liberação Manual do Firewall do Windows para Ping (ICMP)
Por padrão de fábrica, o Windows 10 e 11 descartam pings recebidos. Para permitir que o Central Server realize diagnósticos remotos de ping nesta máquina, execute no **PowerShell como Administrador**:
```powershell
Enable-NetFirewallRule -Name 'FPS-ICMP4-ERQ-In'
```
*(Ou no Prompt de Comando / CMD como Admin):*
```cmd
netsh advfirewall firewall add rule name="Permitir Ping ICMPv4" protocol=icmpv4:8,any dir=in action=allow
```

#### C. Operação da Janela Desktop
- A interface mostra o status da conexão com o servidor (`Conectado` ou `Desconectado`).
- Exibe o Switch LLDP e porta detectados.
- Exibe o status da placa Wi-Fi e nível do sinal.
- O botão *"Executar Teste Agora"* permite disparar a medição imediatamente sem esperar o intervalo automático.

---

### 2.2. No Linux (Debian, Ubuntu, Raspberry Pi)

#### A. Instalação em 1 Comando via Terminal
Execute na máquina cliente com privilégios de `root`:
```bash
curl -fsSL https://nettools.acon.uepg.br/downloads/install-probe.sh | sudo bash
```
*(Caso use IP direto na sua rede local, substitua pelo IP da sua VM: `http://10.0.x.x:8080/downloads/install-probe.sh`)*

#### O que o instalador faz automaticamente:
1. Detecta a arquitetura da máquina (`amd64`, `arm64`, `armhf`).
2. Baixa o binário compilado otimizado `uepg-probe`.
3. Instala o pacote `lldpd` via gerenciador de pacotes para viabilizar a captura de switches de rede no Linux.
4. Registra e ativa o serviço nativo `systemd` (`uepg-probe.service`) para inicializar junto com o sistema operacional.
5. Cria o comando de monitoramento `uepg-probe-monitor`.

#### B. Gerenciando o Serviço Linux
```bash
sudo systemctl status uepg-probe    # Verifica se a sonda está rodando
sudo systemctl restart uepg-probe   # Reinicia a sonda
sudo systemctl stop uepg-probe      # Para a execução temporariamente
sudo journalctl -u uepg-probe -f    # Acompanha logs em tempo real
```

#### C. Monitor Interativo de Terminal (`uepg-probe-monitor`)
Em qualquer computador Linux com a sonda instalada, você pode digitar:
```bash
uepg-probe-monitor
```
Isso abrirá uma interface de terminal colorida com animação de radar Wi-Fi, exibindo a latência atual, perda de pacotes, switch LLDP e dados da placa em tempo real.

---

## 3. Operando o Servidor Central na VM (`central-server`)

O Servidor Central é instalado preferencialmente em uma Máquina Virtual Linux (Debian 11/12 ou Ubuntu Server).

### 3.1. Deploy Automatizado em 1 Comando
Para subir o servidor pela primeira vez ou atualizar a versão na VM:
```bash
# 1. Acesse a pasta do servidor central
cd polling-app-acon/central-server

# 2. Execute o assistente de deploy
chmod +x deploy.sh
sudo ./deploy.sh
```

### 3.2. Comandos de Gerenciamento do Servidor
O script `deploy.sh` conta com parâmetros auxiliares para facilitar o dia a dia do administrador:

| Comando | Função |
| :--- | :--- |
| `./deploy.sh` | Inicialização rápida dos containers Docker (sobe em background). |
| `./deploy.sh --status` | Exibe a saúde do servidor Go, o status do MongoDB e uso de memória. |
| `./deploy.sh --logs` | Acompanha em tempo real o log de todas as sondas enviando telemetria. |
| `./deploy.sh --wizard` | Abre o assistente interativo para reconfigurar portas, IPs e senhas no terminal. |
| `./deploy.sh --down` | Encerra os containers com segurança (mantém todos os dados do banco intactos). |
| `./deploy.sh -y` | Execução não-interativa e silenciosa (ideal para pipelines de CI/CD). |

---

### 3.3. Configuração de Variáveis de Ambiente (`.env`)
No diretório `central-server/`, o arquivo `.env` controla todo o comportamento do núcleo do sistema:

```env
# Porta HTTP exposta pelo Central Server
PORT=8080

# String de Conexão com o MongoDB TimeSeries
MONGO_URI=mongodb://localhost:27017

# Domínio ou IP do Host Central para downloads e Webhooks
APP_DOMAIN=nettools.acon.uepg.br
PUBLIC_SERVER_URL=https://nettools.acon.uepg.br

# IP do Servidor DNS Institucional padrão a ser auditado
DNS_TARGET_DEFAULT=177.101.22.40

# IP do Gateway Core Central do ACON para cálculo de latência de trânsito
CORE_GATEWAY_DEFAULT=10.0.101.1

# Chave secreta interna para validação de sessões e integridade de auto-update
SESSION_SECRET=uepg-acon-secure-jwt-token-2026
```
Ao editar o arquivo `.env`, execute `./deploy.sh` para recarregar as novas variáveis imediatamente.

