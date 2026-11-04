# Dashboard & Gráficos Interativos — UX/UI

A renderização gráfica do sistema é baseada no **Chart.js 4.4**, personalizada com curvas de Bézier suaves, preenchimentos em gradiente translúcido e eixos dinâmicos.

---

## 1. Empilhamento Vertical Responsivo (`.charts-stack-section`)

Anteriormente, os gráficos de throughput e latência eram dispostos lado a lado em colunas assimétricas (`1.5fr 1fr`), o que espremia as séries temporais em resoluções intermediárias e dificultava a leitura de picos de tráfego.

Na arquitetura atual:
- Todos os gráficos são organizados em **pilha vertical** (`display: flex; flex-direction: column; width: 100%;`).
- Cada cartão de gráfico ocupa 100% da largura útil da tela com altura uniforme de 250px.
- Gráficos mantêm espaçamento lateral (`padding.right: 32px`) para garantir que o último ponto temporal (ex: `14:00`) nunca transborde ou corte o rótulo do eixo X.

---

## 2. Gráficos Dedicados

### 2.1. Gráfico de Throughput
- **Métricas:** Download (Cyan `#0284c7`) e Upload (Purple `#4f46e5`).
- **Unidade do Eixo Y:** Megabits por segundo (`Mbps`).
- **Comportamento:** Ocultado automaticamente na visão da sonda caso o perfil atribuído à máquina não meça velocidade de banda.

### 2.2. Gráfico de Latência
- **Métricas:** Ping ao Gateway Local (Emerald `#10b981`).
- **Unidade do Eixo Y:** Milissegundos (`ms`).
- **Escala:** Escala automática ajustada aos tempos médios da rede local.

### 2.3. Gráfico Dedicado de Perda de Pacotes
- **Métricas:** Taxa percentual de pacotes descartados (Rose `#ef4444`).
- **Unidade do Eixo Y:** Porcentagem (`%`).
- **Escala Fixa:** Limites configurados rigorosamente entre **0%** e **100%** para representar visualmente a integridade física do enlace sem distorções de zoom.

---

## 3. Adaptação a Temas e Redimensionamento

Ao alternar entre os modos Claro e Escuro, a função `updateChartsTheme()` atualiza dinamicamente as cores de grid, rótulos e caixas de tooltip sem recriar as instâncias dos gráficos (`chart.update('none')`), preservando as animações e o estado do usuário.

