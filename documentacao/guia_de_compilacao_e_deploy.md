# 🚀 Guia de Compilação, Instalação e Deploy

Este guia orienta o desenvolvedor ou administrador na compilação do servidor central, geração de instaladores e distribuição das sondas clientes.

---

## 1. Pré-Requisitos

- **Compilador Go**: Versão 1.22 ou superior instalada ([golang.org](https://go.dev/dl/)).
- **Git**: Para versionamento e controle das branches.
- **Docker & Docker Compose** (para o Central Server em produção/VM).
- **NSIS** (opcional no Windows, caso deseje empacotamento com instalador avançado).

---

## 2. Compilação e Execução do Central Server

### 2.1. Via Docker Compose (Recomendado para Servidores/VMs)
No diretório `central-server/`:

```bash
cd central-server

# Build sem cache para garantir inclusão de todos os binários e assets estáticos
docker compose build central-server --no-cache

# Inicializa banco de dados e servidor em background
docker compose up -d

# Visualização de logs
docker compose logs -f central-server
```

### 2.2. Execução Local para Desenvolvimento (Windows ou Linux)
```bash
cd central-server
go run .
```
O servidor estará acessível em `http://localhost:8080`.

---

## 3. Compilação do Client Probe (Windows)

No diretório `client-probe/`:

### 3.1. Gerar Executável e Instalador Oficial
Execute o script PowerShell fornecido:
```powershell
cd client-probe
.\gerar-instalador.ps1
```
Ou no prompt de comando tradicional:
```cmd
gerar-instalador.bat
```

**O que este script faz:**
1. Sincroniza os arquivos de interface web do frontend.
2. Compila o binário de produção `build\bin\uepg-net-tools.exe` com flag `-H windowsgui` (sem janela preta de prompt do DOS).
3. Compila o assistente nativo de instalação `build\bin\UEPG-Net-Tools-Instalador.exe`.
4. Copia os binários diretamente para a pasta pública de downloads do servidor central (`central-server\public\downloads\`).

---

## 4. Compilação do Client Probe (Linux / Debian / Raspberry Pi)

No diretório `client-probe/`:

### 4.1. Compilação Cruzada dos Binários Linux
```powershell
cd client-probe
.\compilar-linux.ps1
```
Ou via terminal Bash no Linux:
```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags cli -ldflags="-s -w" -o build/linux/uepg-probe-linux-amd64 .
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags cli -ldflags="-s -w" -o build/linux/uepg-probe-linux-arm64 .
```

Os binários compilados são publicados em `central-server/public/downloads/`.

---

## 5. Instalação em Novas Máquinas Clientes

### 5.1. Instalação no Windows
1. Acesse o Central Server pelo navegador em qualquer máquina:  
   `https://seu-servidor-central/`
2. Na barra de navegação ou aba de downloads, baixe o **UEPG-Net-Tools-Instalador.exe**.
3. Execute o instalador:
   - Selecione a pasta de instalação (padrão: `C:\Program Files\UEPG-Net-Tools` ou `C:\ProgramData\UEPG-Net-Tools`).
   - Marque se deseja atalho na Área de Trabalho e inicialização automática com o Windows.
   - Clique em **Instalar**.
4. A sonda inicia imediatamente e conecta-se ao servidor central para receber o perfil de medição da máquina.

### 5.2. Instalação no Linux via Script Dinâmico
Em servidores Debian/Ubuntu ou Raspberry Pi, execute uma única linha no terminal:
```bash
curl -fsSL https://seu-servidor-central/downloads/install-probe.sh | sudo bash
```
O script configura o binário em `/opt/uepg-probe/`, cria o serviço systemd `uepg-probe.service` e inicia o monitoramento contínuo automaticamente.
