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
- [ ] **8. Tema Claro & Tema Escuro:** O contraste de texto e fundos das tabelas foi verificado em ambos os modos.
- [ ] **9. Clique na Logo:** Clicar na logo da UEPG no canto superior esquerdo reseta a máquina individual, o filtro de grupo e volta ao Dashboard consolidado.

