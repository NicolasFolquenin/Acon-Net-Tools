# 🚀 Guia de Testes Prévios e Lançamento de Atualização das Sondas (`client-probe`)

Este documento orienta como validar as novas funcionalidades localmente e, quando a equipe estiver pronta, como lançar a nova versão para que **todas as máquinas clientes instaladas no parque da universidade se atualizem automaticamente via auto-update**, sem necessidade de intervenção física ou manual.

---

## 🧪 Parte 1: Realizando os Testes Locais (Antes de Alterar a Versão)

Antes de gerar a versão oficial e distribuir para o parque, valide as novas funcionalidades na sua máquina de desenvolvimento:

### 1. Testar a Sonda Desktop Localmente
No terminal ou clicando duas vezes pelo Windows Explorer:
```cmd
cd client-probe
testar.bat
```
> O script `testar.bat` sincroniza os arquivos da interface (`frontend/` $\rightarrow$ `frontend/dist/`), compila o executável de teste em `build/test-runner.exe` e abre a janela gráfica do aplicativo.

### 2. O que Validar nos Testes:
* **Usuário Ativo no Cabeçalho:** Verifique se o badge no topo exibe o seu usuário logado no formato `Usuário: <seu_login>`.
* **Bloco 5 - Contas de Usuário & Sessões do Sistema:**
  * Logo abaixo do inventário de placas de rede, confira o card de resumo com o **Usuário da Medição (Processo Sonda)** e **Usuário Ativo no Console (Área de Trabalho)**.
  * Veja se a tabela lista os usuários locais da máquina (ex: contas de administrador, contas de usuário comum) e se a sua conta atual está marcada com badge verde **Logado**.
* **Envio para o Central Server:**
  * Clique no botão **"Rodar Teste Agora"**.
  * Verifique se as medições são enviadas com sucesso (`Sincronizado com o servidor central`).

---

## 🔄 Parte 2: Como Funciona o Auto-Update nas Máquinas Instaladas

Cada sonda em execução nas máquinas remotas possui uma rotina de auto-update contínua:
1. **Verificação periódica:** A cada 30 minutos (e também na inicialização ou ao clicar no botão de atualizar no cabeçalho), a sonda consulta `GET /api/v1/download/version` no Central Server.
2. **Comparação de versão:** A sonda compara a versão retornada pelo servidor com a sua versão interna (`CurrentClientVersion`).
3. **Download e Substituição transparente (Swap Atômico):**
   * Se o servidor tiver uma versão superior (ex: `v2.5.0` > `v2.4.1`), a sonda baixa o novo executável em segundo plano (`.new`).
   * No Windows, ela renomeia o executável em execução para `.old` e move o `.new` para o lugar oficial.
   * O novo processo é iniciado em segundo plano e o processo antigo encerra.
   * Na próxima inicialização, o arquivo `.old` é excluído automaticamente.

---

## 📦 Parte 3: Passo a Passo para Lançar a Nova Versão

Quando os testes forem concluídos e a equipe aprovar o lançamento:

### Passo 1: Incrementar a Versão da Sonda
Defina o novo identificador de versão (por exemplo, `2.5.0`):

1. No arquivo [`client-probe/updater.go`](../../client-probe/updater.go):
   ```go
   // Altere de "2.4.1" para a nova versão:
   const CurrentClientVersion = "2.5.0"
   ```

2. No arquivo [`client-probe/frontend/index.html`](../../client-probe/frontend/index.html):
   ```html
   <!-- Atualize o badge de versão no cabeçalho -->
   <span class="header-version-badge" id="client-version-badge" title="Versão da Sonda Local">v2.5.0</span>
   ```

3. No arquivo [`central-server/public/downloads/version.txt`](../../central-server/public/downloads/version.txt):
   ```text
   2.5.0
   ```

---

### Passo 2: Compilar os Binários de Produção

No PowerShell, na pasta `client-probe`:

#### A. Para Windows (Executável puro para auto-update e Instalador):
```powershell
cd client-probe
.\gerar-instalador.ps1
```
> Esse script compila o `uepg-net-tools.exe` com ícone embutido e flags de produção do Wails, gera o instalador `UEPG-Net-Tools-Instalador.exe` e copia ambos para `central-server/public/downloads/`.

#### B. Para Linux (se houver mini PCs com Linux):
```powershell
.\compilar-linux.ps1
```

---

### Passo 3: Publicar no Servidor Central em Produção

No servidor onde o `central-server` está hospedado (ex: servidor institucional ou container Docker):

1. **Atualizar o código do Central Server:**
   ```bash
   git pull origin feat/probe-isolated-dynamic-metrics # ou branch principal aprovada
   ```
2. **Garantir que os arquivos estão na pasta `public/downloads/` do servidor:**
   * `public/downloads/version.txt` (conteúdo: `2.5.0`)
   * `public/downloads/uepg-net-tools.exe` (novo binário compilado)
   * `public/downloads/UEPG-Net-Tools-Instalador.exe` (novo instalador completo)
3. **Reiniciar o serviço do Central Server:**
   ```bash
   # Se rodar como serviço systemd:
   sudo systemctl restart central-server
   
   # Ou se rodar via Docker:
   docker compose restart central-server
   ```

---

### Passo 4: Acompanhar a Atualização pelo Dashboard Web

Após publicar a nova versão no Central Server, você não precisa fazer nada nas máquinas dos laboratórios.

1. Abra o Dashboard do Central Server no navegador.
2. Na tabela **Computadores & Sondas Monitoradas**:
   * As máquinas que ainda estiverem na versão anterior exibirão o badge de aviso em laranja: `v2.4.1 (Atualização Pendente)`.
   * Conforme cada máquina atingir o ciclo de 30 minutos e reiniciar automaticamente, o badge mudará para verde: `v2.5.0 (Atualizado)`.
   * Ao focar na máquina, você verá imediatamente o **usuário logado** e a lista completa de contas locais daquele computador.

---

## 🆘 Troubleshooting & Atualização Manual de Emergência

Se alguma máquina estiver com a conexão de rede oscilando ou o auto-update for bloqueado por antivírus institucional:

1. **Forçar atualização via interface gráfica:**
   * Na tela da sonda, clique no botão de atualização (ícone de seta circular ao lado de `v2.4.1`).
   * A sonda consulta o servidor imediatamente e aplica o patch na hora.

2. **Atualização rápida via PowerShell (sem reinstalar):**
   Execute no PowerShell da máquina cliente como Administrador:
   ```powershell
   Stop-Process -Name "uepg-net-tools" -Force -ErrorAction SilentlyContinue
   $server = "https://akvorado.acon.dev.uepg.br"
   $dest = "$env:ProgramData\UEPG-Net-Tools\uepg-net-tools.exe"
   Invoke-WebRequest -Uri "$server/api/v1/download/binary/windows" -OutFile $dest
   Start-Process -FilePath $dest -WindowStyle Hidden
   ```
