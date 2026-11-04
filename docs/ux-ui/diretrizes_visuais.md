# Diretrizes Visuais & Design System

Tanto o **Dashboard Web NOC** quanto a **Interface Desktop Wails** compartilham os mesmos fundamentos de design, garantindo coerência visual, clareza operacional e legibilidade técnica em ambientes de NOC.

---

## 1. Princípios de Design

1. **Eficiência para Operação NOC:** Informações vitais (status de nó, perda de pacotes, alarmes e saturação de banda) recebem contraste imediato através de cores semânticas.
2. **Dual-Theme Nativo (Escuro & Claro):** Suporte completo ao modo noturno (Dark Mode) de baixo brilho para centros de monitoramento e modo diurno (Light Mode) de alto contraste.
3. **Tipografia Técnica Especializada:**
   - **`Inter`:** Utilizada para títulos, textos de navegação, modais e rótulos contextuais.
   - **`JetBrains Mono`:** Utilizada exclusivamente para dados de rede (endereços IPv4/IPv6, MAC address, nomes de switches, portas de rede, valores numéricos e badges de versão).

---

## 2. Paleta Semântica de Cores

| Função Semântica | Token CSS | Cor Modo Escuro | Cor Modo Claro | Uso |
| :--- | :--- | :--- | :--- | :--- |
| **Download / Throughput** | `--cyan-primary` | `#06b6d4` / `#22d3ee` | `#0284c7` | Gráficos e cards de taxa de download |
| **Upload** | `--purple-primary` | `#6366f1` / `#818cf8` | `#4f46e5` | Gráficos e cards de taxa de upload |
| **Latência Normal / OK** | `--emerald-primary`| `#10b981` / `#34d399` | `#059669` | Ping gateway excelente (< 5ms) e nós online |
| **Perda de Pacotes / Erro**| `--rose-primary` | `#f43f5e` / `#ef4444` | `#e11d48` | Gráfico de perda de pacotes e alarmes |
| **Atenção / Degradado** | `--amber-primary` | `#f59e0b` / `#fbbf24` | `#d97706` | Sondas desatualizadas e latência elevada |
| **Versão / Software** | `--sky-primary` | `#38bdf8` | `#0284c7` | Badges de versão da sonda (`v2.4.1`) |

---

## 3. Badges Semânticos Padronizados

- **Badges de Versão:**
  - Atualizada: fundo azul claro translúcido (`rgba(56, 189, 248, 0.12)`), borda sutil e texto em destaque.
  - Desatualizada: fundo âmbar translúcido alertando discrepância com o servidor.
- **Badges de Interface de Rede:**
  - Ethernet: Azul (`.badge-iface-type.eth`).
  - Wi-Fi: Verde esmeralda (`.badge-iface-type.wifi`).
  - Virtual: Cinza suave (`.badge-iface-type.virtual`).
  - Rota Principal: Âmbar arredondado (`.badge-iface-priority`).

