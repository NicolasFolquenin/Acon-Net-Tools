# 🎨 Diretrizes de UX e UI - Padrão Visual Institucional UEPG Net Tools

Este documento estabelece o **padrão de interface (UI)** e **experiência do usuário (UX)** obrigatório para todo o ecossistema do **UEPG Net Tools**. Qualquer futura modificação, tela nova ou refatoração visual deve **obrigatoriamente seguir este guia**.

---

## 🚫 1. Regra Fundamental: Proibição de Emojis

**Emojis estão estritamente proibidos em toda a interface do sistema.**
- Emojis têm renderização inconsistente entre sistemas operacionais (Windows, Linux, macOS, Android), parecem amadores em ambientes corporativos/acadêmicos e quebram alinhamento vertical de linhas e fontes monospace.
- **Substitutos Obrigatórios:**
  - Em vez de emojis, utilize **ícones vetoriais SVG** padronizados (stroke: 2, fill: none, dimensões 12px a 16px).
  - Para status de conexão, utilize **badges com status dots** em CSS puro (`.status-indicator-dot`).
  - Para destaque tipográfico, use **badges textuais limpas** (`.badge`, `.pill`, `.tag`).

---

## 📐 2. Padrão Universal para Tabelas e Listas de Dados

Toda e qualquer lista de dados (Sondas, Máquinas, Grupos, Módulos, Usuários, etc.) deve **seguir a mesma estrutura arquitetural**:

```
┌────────────────────────────────────────────────────────────────────────┐
│ CARD PRINCIPAL DA TELA (.page-header-row)                             │
│ Título da Seção | Subtítulo explicativo | Botões de Ação Global        │
├────────────────────────────────────────────────────────────────────────┤
│ BARRA DE FILTROS (.filters-card)                                       │
│ [🔍 Busca Textual] [Dropdown 1] [Dropdown 2] [Filtro Data] [Contador] │
├────────────────────────────────────────────────────────────────────────┤
│ CONTAINER DA TABELA COM OVERFLOW CONTROLADO (.table-responsive)        │
│ ┌────────────────────────────────────────────────────────────────────┐ │
│ │ CABEÇALHO PADRÃO (THEAD): 100% largura, uppercase, muted, fixo     │ │
│ ├────────────────────────────────────────────────────────────────────┤ │
│ │ CORPO DA TABELA (TBODY): Linhas alternáveis, hover sutil, cells    │ │
│ ├────────────────────────────────────────────────────────────────────┤ │
│ │ AÇÕES NA ÚLTIMA COLUNA: Compactas, alinhadas à direita, sem emojis │ │
│ └────────────────────────────────────────────────────────────────────┘ │
├────────────────────────────────────────────────────────────────────────┤
│ RODAPÉ DE PAGINAÇÃO (.table-pagination-footer)                        │
│ "Exibindo X a Y de Z • N por página"     [← Anterior] Pág 1 de N [→]   │
└────────────────────────────────────────────────────────────────────────┘
```

### ⏱️ Ordenação Padrão por Recência de Sinal
- **Último Sinal Mais Próximo Primeiro:** Toda lista de máquinas e sondas (tanto no Dashboard de Telemetria quanto no Gerenciamento de Máquinas) deve obrigatoriamente ordenar os registros de forma decrescente pelo timestamp do último sinal (`timeB - timeA`). As máquinas com telemetria mais recente (ex.: há poucos segundos, há 8 min) devem aparecer no topo da tabela, e as máquinas inativas ou sem sinal há mais tempo (ex.: há 138h) devem ficar no final.
- **Desempate:** Em caso de empate ou ausência de sinal, utiliza-se a ordem alfabética do hostname.

### 🔎 Busca Multifuncional Unificada
A caixa de busca textual (`machines-search-input` e `nodes-search-input`) não deve se limitar apenas ao hostname. Ela deve cobrir:
- **Hostname / Nome da Máquina**
- **Endereço IP** (IPv4 e IPv6)
- **Endereço MAC** (com tratamento automático para aceitar formatos com dois pontos `00:11:22`, hifens `00-11-22` ou contínuo `001122`)
- **Switch e Porta** (ex.: `ACONS0001`, `GE1/0/4`, `0/1`)
- **VLAN**
- **Grupo Vinculado** (tanto o identificador técnico `group_id` quanto o nome legível institucional `grp.name`)
- **Localização / Departamento** (`location`)
- **Módulos / Testes Ativos** (ex.: `ping`, `speedtest`, `traceroute`)
- **Status Textual** (`online`, `sem sinal`)

---

## 🔒 3. Prevenção de Quebra de Layout e Elementos Vazando

Para evitar que colunas e botões extrapolem o container da página (como acontecia na aba de máquinas):

1. **Container Responsivo Obrigatório:**
   Toda tabela deve estar envolvida em uma `div` com a classe `.table-responsive`:
   ```css
   .table-responsive {
     width: 100%;
     overflow-x: auto;
     -webkit-overflow-scrolling: touch;
   }
   ```
2. **Largura Total e Borda do Cabeçalho:**
   A tabela deve possuir `width: 100%`, `border-collapse: collapse` e seus elementos `th` e `td` devem ter larguras proporcionais ou `white-space: nowrap` estritamente nas colunas que exigem (como IP/MAC e datas).
3. **Limitação de Elementos Rígidos:**
   - Dropdowns (`<select>`) dentro de linhas de tabela devem ter largura máxima controlada:
     ```css
     max-width: 170px;
     text-overflow: ellipsis;
     white-space: nowrap;
     overflow: hidden;
     ```
   - Colunas de **Ações** não devem ultrapassar `220px`. Devem agrupar botões compactos (`btn-sm` ou `btn-icon`) com ícone SVG e tooltip explicativo, sem botões esticados que quebrem a linha.

---

## 🧭 4. Padrão de Navegação e Identidade Institucional

- **Logo da UEPG (Canto Superior Esquerdo):**
  - Clicar na logo da UEPG ou no nome institucional no topo deve **sempre retornar à visão consolidada do Dashboard**, limpando seleções individuais de sondas (`selectedChartNode = 'all'`) e restaurando a visão geral.
  - O cursor do mouse deve ser `pointer` e fornecer animação sutil ao passar o cursor.

---

## 🎨 5. Contraste em Modais & Contorno em Cards de Módulos

1. **Modais de Detalhes (ex.: Host/PC, Configurações):**
   - Nunca utilizar cores duras hardcoded escuras (`rgba(15, 23, 42, 0.45)`) com textos acinzentados apagados (`#94a3b8`, `#64748b`).
   - Todos os textos de telemetria, títulos de subseções e nomes de hardware devem utilizar variáveis de contraste semânticas (`var(--text-main)`, `var(--text-muted)`).
   - Dados de status e interfaces devem utilizar badges ou estados visuais com contraste nítido em ambos os temas (Dark & Light). Textos como "Inativa", "Sem IP" ou "Secundária" devem ter legibilidade clara, sem parecer que estão apagados ou inexistentes.
   - **Modal de Detalhes Sem Gráficos Embutidos:** O modal acionado na tabela de máquinas apresenta exclusivamente as **principais informações e estado atual** da sonda (KPIs mais recentes, telemetria de hardware/SO, conectividade LLDP, último sinal completo e módulos ativos). **Gráficos pesados não devem ser renderizados dentro deste modal**. O histórico detalhado de séries temporais é centralizado no Dashboard, acionado pelo botão **"Focar no Dashboard"**.

2. **Cards de Módulos & Catálogo JSON:**
   - As caixas de módulos (`.module-card`) devem possuir contorno aparente (`1.5px solid var(--border-card-prominent)`) e elevação com sombra suave (`box-shadow: 0 2px 10px rgba(0, 0, 0, 0.08)`), para nunca se mesclarem ao fundo da página.
   - No tema claro (`[data-theme="light"]`), o contorno deve ter espessura de `1.5px solid #94a3b8` para perfeita separação dos cards em relação ao fundo branco.
   - **Proibição de Exclusão no Frontend:** Módulos do sistema e medições registradas não devem exibir opção de exclusão no frontend. Os cards devem conter apenas ações de visualização/edição de parâmetros e download do arquivo `.json`.

3. **Gráficos Comparativos & Séries Temporais:**
   - Para evitar estouro horizontal em qualquer resolução ou corte lateral, os gráficos comparativos principais devem ser dispostos **empilhados verticalmente** (`display: flex; flex-direction: column; width: 100%; gap: 24px;`).
   - Cada painel de gráfico (`.chart-panel`) e contêiner do canvas (`.chart-canvas-container`) deve conter estritamente `min-width: 0; max-width: 100%; box-sizing: border-box; overflow: hidden;` para que o Chart.js responda fluidamente à largura útil do contêiner.
   - **Filtro por Grupo no Dashboard:** O cabeçalho dos gráficos comparativos deve fornecer um seletor `<select id="chart-group-filter-select">` permitindo filtrar as séries por grupo de máquinas cadastrado.
   - **Botão de Limpar e Indicador de Foco:** Quando um grupo específico estiver selecionado, deve ser exibido um botão de ação rápida "Limpar Filtro" e a badge `.chart-focus-pill` deve sinalizar `Grupo: [Nome] (N máquinas)`. Clicar na logo da UEPG ou no botão de limpar deve restaurar a visão global de todas as máquinas.

---

## ✅ 6. Checklist Obrigatório Pré-Deploy (Passo a Passo)

Antes de aprovar qualquer alteração visual no Central Server, verifique cada item abaixo:

- [ ] **1. Verificação de Emojis:** Não há emojis no HTML ou no JS gerado dinamicamente (usar apenas ícones SVG e texto sóbrio).
- [ ] **2. Contenção no Card:** Redimensione a janela do navegador para 1280px e 1024px. Nenhum botão ou cabeçalho vaza para fora do card branco/escuro.
- [ ] **3. Barra de Rolagem Horizontal Interna:** Se a tabela for mais larga que a tela, a barra de rolagem horizontal surge **dentro** do card da tabela, e não na página inteira.
- [ ] **4. Barra de Filtros e Busca Multifuncional:** A barra de busca filtra instantaneamente por Hostname, IP, MAC (independente de hífen/dois-pontos), Switch, Porta, Grupo e Status.
- [ ] **5. Ordenação por Último Sinal:** Máquinas com sinal mais recente/próximo aparecem no topo por padrão em ambas as visões de lista.
- [ ] **6. Filtro por Grupo em Gráficos:** O seletor de grupo nos gráficos comparativos filtra as séries apenas das máquinas do grupo selecionado, exibe a contagem de máquinas no pill e o botão "Limpar Filtro" restaura o estado global.
- [ ] **7. Paginação Operacional:** O rodapé de paginação mostra itens por página, controles `Anterior` / `Próxima` e indicador de página atual.
- [ ] **8. Modal de Detalhes Leve:** O modal de detalhes da máquina abre de forma instantânea com dados essenciais (KPIs, SO, LLDP, módulos e data do último sinal), sem gráficos embutidos, redirecionando o histórico para o Dashboard via "Focar no Dashboard".
- [ ] **9. Tema Claro & Tema Escuro:** O contraste de texto e fundos das tabelas foi verificado em ambos os modos.
- [ ] **10. Clique na Logo:** Clicar na logo da UEPG no canto superior esquerdo reseta a máquina individual, o filtro de grupo e volta ao Dashboard consolidado.



---

## 7. Modo Isolado de Sonda (Visualização de Máquina Única)

Quando um operador ou administrador clica em uma máquina individual para isolar seus gráficos no Dashboard, as seguintes diretrizes são de aplicação mandatória:

### 7.1. Eliminação Estrita de Redundâncias
- **Ocultação da Tabela de 1 Linha:** A seção "Computadores & Sondas Monitoradas" deve ser automaticamente ocultada (`display: none`), pois exibir uma tabela de computadores contendo apenas a máquina que já está selecionada polui a visão do operador.
- **Remoção do Filtro de Grupos:** O seletor de grupo dos gráficos comparativos deve ser ocultado no modo focado, pois não faz sentido filtrar por grupo se o operador está analisando exclusivamente as séries temporais de um nó específico.

### 7.2. Substituição Contextual do Primeiro Card KPI (Perda de Pacotes Gateway)
- Em vez de exibir um card trivial "Sondas Monitoradas: 1", o primeiro card do topo deve se transformar dinamicamente em **"Perda de Pacotes Gateway"**.
- Deve exibir a porcentagem de perda aferida (ex.: `0.0%`), badge de integridade (`0% Perda` em verde, `<= 5%` em amarelo, `> 5%` em vermelho) e o alvo de gateway local testado via ICMP.
- Ao retornar para a visão consolidada de todas as sondas, o card restaura automaticamente o estado "Sondas Monitoradas" e contagem global.

### 7.3. Painel Técnico Rico Embutido na Página
Em vez de esconder os detalhes do computador em um modal que tampa a visão dos gráficos, a página embutirá o container `#single-machine-details-panel` contendo quatro blocos especializados:
1. **Status Wi-Fi & Potência do Sinal:**
   - Detecção se o computador possui adaptador wireless e se está associado a uma rede.
   - Em caso positivo: barra de intensidade de sinal com preenchimento colorido, porcentagem precisa, potência em dBm, SSID da rede, BSSID/AP, canal, faixa de frequência (2.4GHz / 5GHz), rádio (802.11ax/ac/n) e taxas Tx/Rx.
   - Em caso de conexão por cabo Ethernet: notificação visual clara informando conexão cabeada ativa sem poluir com dados wireless vazios.
2. **Módulos Ativos Carregados:**
   - Chips visuais com os testes agendados da sonda (Vazão Download/Upload, Ping Gateway, Ping Core NTI, Perda de Pacotes, LLDP, Wi-Fi, DNS).
   - Indicação se a sonda herda o perfil do grupo ou se possui testes customizados adicionais.
3. **Topologia Switch LLDP & Conexão de Rede:**
   - Nome do switch corporativo, porta física, VLAN operacional e protocolo.
   - Botão de ação destacada para inspecionar vizinhos conectados no mesmo equipamento.
4. **Inventário de Placas de Rede & MACs Físicos:**
   - Tabela responsiva com contador total de placas, status dot (Ativa/Inativa), descrição do hardware, badge de tipo (Ethernet, Wi-Fi, Virtual), endereço MAC físico formatado, endereços IP e identificação da rota principal/gateway.

### 7.4. Múltiplos Destinos de Latência no Gráfico
- O gráfico de latência individual deve disponibilizar controles segmentados:
  - `[Gateway]`: Medição contínua para o Gateway local.
  - `[Core NTI]`: Medição contínua para o Core institucional no anel óptico.
  - `[Ambos]`: Plotagem comparativa simultânea com curvas distintas (#10b981 e #3b82f6) e legendas para análise de gargalos (se o salto local ou metropolitano está degradado).

### 7.5. Comparação de Máquinas Conectadas no mesmo Switch LLDP
- Através do modal de switch LLDP (`#modal-switch-machines`), o operador pode visualizar todos os computadores conectados na mesma infraestrutura física.
- O botão **"Comparar nos Gráficos"** filtra as séries temporais comparativas exclusivamente para essas máquinas vizinhas, permitindo identificar se oscilações de banda e latência são pontuais do host ou generalizadas naquele switch/setor.
