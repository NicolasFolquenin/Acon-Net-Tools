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

## ✅ 5. Checklist Obrigatório Pré-Deploy (Passo a Passo)

Antes de aprovar qualquer alteração visual no Central Server, verifique cada item abaixo:

- [ ] **1. Verificação de Emojis:** Não há emojis (`📊`, `⚙️`, `⚡`, `🗑`, `✔`, `✖`) no HTML ou no JS gerado dinamicamente.
- [ ] **2. Contenção no Card:** Redimensione a janela do navegador para 1280px e 1024px. Nenhum botão ou cabeçalho vaza para fora do card branco/escuro.
- [ ] **3. Barra de Rolagem Horizontal Interna:** Se a tabela for mais larga que a tela, a barra de rolagem horizontal surge **dentro** do card da tabela, e não na página inteira.
- [ ] **4. Barra de Filtros:** Todos os selects, inputs de busca e contadores estão alinhados e utilizam as classes padrão (`.form-input-sm`).
- [ ] **5. Paginação Operacional:** O rodapé de paginação mostra itens por página, controles `Anterior` / `Próxima` e indicador de página atual.
- [ ] **6. Tema Claro & Tema Escuro:** O contraste de texto e fundos das tabelas foi verificado em ambos os modos.
- [ ] **7. Clique na Logo:** Clicar na logo da UEPG no canto superior esquerdo reseta a máquina selecionada e volta ao Dashboard.
