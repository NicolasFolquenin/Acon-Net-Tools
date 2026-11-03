# Modo Desktop Gráfico (Wails v2) — Sonda Cliente

Nas estações de trabalho com Windows, a sonda opera como um aplicativo de desktop moderno empacotado através do framework **Wails v2** (`github.com/wailsapp/wails/v2`).

---

## 1. Arquitetura e Comunicação IPC

Diferente do Electron que embuti instâncias pesadas do Chromium e Node.js consumindo centenas de megabytes de memória RAM, o Wails v2:
- Utiliza o **Microsoft Edge WebView2** já nativo do Windows 10 e 11.
- Comunica a camada gráfica (HTML5/CSS3/JavaScript) diretamente com o binário em Go através de chamadas IPC assíncronas geradas em tempo de compilação.
- Mantém o executável final enxuto e com consumo inferior a 50 MB de memória.

---

## 2. Estrutura do Frontend Desktop (`client-probe/frontend`)

- **`index.html`:** Layout estruturado com topo de controle, cards numéricos de KPI, painel técnico de hardware, switch e Wi-Fi, gráficos empilhados verticais e tabela de execuções paginada.
- **`style.css`:** Folha de estilos completa com design system dual-theme (Dark/Light), fontes Inter e JetBrains Mono e efeitos de glassmorphism.
- **`main.js`:** Lógica de manipulação de DOM, bindings do Wails (`window.go.main.App`), gerenciamento de tema e renderização dos gráficos Chart.js.
- **`dist/`:** Pasta onde os arquivos estáticos prontos para compilação são distribuídos e empacotados pelo Wails.

---

## 3. Empacotamento e Compilação

Para compilar a aplicação Wails no ambiente Windows:

```powershell
# 1. Sincronizar arquivos para a pasta dist
cd client-probe
Copy-Item frontend\index.html frontend\dist\index.html -Force
Copy-Item frontend\main.js frontend\dist\main.js -Force
Copy-Item frontend\style.css frontend\dist\style.css -Force

# 2. Compilar via Wails CLI ou script bat
wails build -tags desktop,production
# ou executar testar.bat
.\testar.bat
```

O binário resultante incorpora todos os assets estáticos no executável, sem dependência de servidores web locais ou arquivos externos.

