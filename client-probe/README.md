# UEPG-Net-Tools (Client Desktop Wails v2 - Windows)

Agente de monitoramento contínuo de throughput e latência de rede desenvolvido em **Go 1.22+** e **Wails v2**, projetado para rodar em mini PCs e servidores locais no Windows.

---

## 🚀 Funcionalidades

- **Agendador em Segundo Plano:** Goroutine nativa em Go que executa testes de rede a cada 1 hora.
- **Disparo Manual:** Botão "Rodar Teste Agora" com feedback visual de carregamento (spinner e estados de bloqueio).
- **Medição de Rede Realista:**
  - **Latência:** Ping para o gateway padrão via ICMP nativo / TCP probe.
  - **Download & Upload:** Throughput medido em Mbps contra o servidor central.
- **Persistência Local (SQLite):** Gravação em `probe_local.db` para funcionamento e visualização offline das últimas 24 execuções.
- **Sincronização Segura HTTPS:** `POST /api/v1/metrics` para o Servidor Central com autenticação `Authorization: Bearer <api_token>`. Falhas de rede não travam a aplicação.
- **Frontend Moderno:** Tema Dark Slate com destaques em Azul Ciano e Roxo, gráficos interativos com **Chart.js** e modal de configurações dinâmicas.

---

## 📋 Pré-requisitos para Desenvolvimento / Build

1. **Go 1.22+** instalado ([golang.org](https://go.dev/dl/))
2. **Wails v2 CLI:**
   ```powershell
   go install github.com/wailsapp/wails/v2/cmd/wails@latest
   ```
3. **WebView2 Runtime** (já incluído nativamente no Windows 10/11)

---

## ⚙️ Configuração (`config.json`)

O arquivo `config.json` fica no mesmo diretório do executável:

```json
{
  "node_id": "minipc-dc-01",
  "location": "Guarapuava-DC1",
  "server_url": "http://localhost:8080",
  "api_token": "super-secret-probe-token-2026",
  "interval_minutes": 60,
  "ping_target": "192.168.1.1"
}
```

- As opções também podem ser ajustadas em tempo de execução pelo botão de configurações na interface gráfica.

---

## 🛠️ Como Executar e Testar

### 1. Teste Rápido com Live Reload (Sem Reinstalar)
Para testar alterações na hora sem precisar recompilar ou reinstalar:
- **Pelo Windows Explorer:** Basta dar duplo clique no script `testar.bat`.
- **Ou pelo Terminal:**
  ```powershell
  cd client-probe
  wails dev
  ```
> 💡 **Live Reload:** Qualquer alteração salva nos arquivos `frontend/` (HTML, CSS, JS) atualiza instantaneamente na janela do aplicativo aberto, sem precisar fechar nem reinstalar.

---

### 2. Gerar Executável ou Instalador Oficial Windows (`.exe`)

- **Com 1 clique:** Dê duplo clique em `gerar-instalador.bat`.
- **Ou pelo Terminal:**
  ```powershell
  # Apenas o executável portátil:
  wails build -clean -platform windows/amd64

  # Instalador Oficial Windows (Setup .exe com assistente e atalhos):
  wails build -clean -platform windows/amd64 -nsis
  ```
> 📌 *Dica para o Instalador Oficial:* Para gerar o instalador com assistente (.exe instalável), certifique-se de ter o NSIS instalado (`winget install NSIS.NSIS`).
> Os arquivos finais ficam na pasta `client-probe/build/bin/`.

---

## 🗄️ Estrutura do Banco Local SQLite (`probe_local.db`)

Tabela `metrics`:
- `id` (INTEGER PRIMARY KEY)
- `node_id` (TEXT)
- `location` (TEXT)
- `timestamp` (DATETIME)
- `download_mbps` (REAL)
- `upload_mbps` (REAL)
- `ping_gateway` (REAL)
- `synced` (INTEGER 0 ou 1)
- `error_message` (TEXT)
