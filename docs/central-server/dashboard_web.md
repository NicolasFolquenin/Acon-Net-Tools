# Dashboard Web NOC — Servidor Central

O Dashboard NOC (`central-server/public`) é a interface central de visualização e controle operacional da rede. Ele opera diretamente no navegador sem frameworks pesados, garantindo carregamento instantâneo e baixo consumo de recursos na VM.

---

## 1. Modos de Visualização

O painel oferece dois modos principais de operação:

### 1.1. Visão Geral (Agregada)
- Exibe o resumo de todos os nós conectados na rede.
- Tabela com status em tempo real: **Nome da Máquina**, **IP**, **Localização**, **Grupo**, **Versão da Sonda**, **Switch LLDP**, **Porta**, **VLAN**, **Latência** e **Throughput**.
- **Badges de Versão do Software:**
  - `v2.4.1 (Atualizada)`: Fundo azul suave com borda e texto destacado.
  - `v2.4.0 (Desatualizada)`: Destaque em âmbar alertando a necessidade de auto-update da sonda.
- Filtros por grupo, pesquisa rápida por hostname/MAC e ordenação por latência ou perda de pacotes.

### 1.2. Visão Isolada de Sonda (Single Machine Drilldown)
Ao clicar em qualquer máquina, o operador acessa um painel dedicado com foco exclusivo no nó:
- **Cabeçalho com Identidade Completa:** Hostname em fonte mono, IP, Grupo ativo, VLAN, Switch e Badge da versão da sonda (`#sm-version-badge`).
- **Cards de Métricas Dinâmicos:** Exibe somente as métricas que o perfil da máquina de fato mede.
  - Card de **Perda de Pacotes Gateway** (com indicador de % de perda e alvo).
  - Cards de **Throughput Download** e **Upload** (ocultados automaticamente se a sonda estiver em perfil leve).
  - Card de **Latência Gateway Local**.
- **Painel Técnico de Hardware, Wi-Fi & Switch:**
  - **Status Wi-Fi & Potência:** Medidor de sinal com barra gráfica de percentual e dBm quando conectada via rádio, ou aviso de enlace exclusivo quando em par trançado (Ethernet).
  - **Módulos Ativos Carregados:** Pílulas visuais de cada teste programado para o nó.
  - **Topologia Switch LLDP:** Nome do equipamento, porta de enlace, VLAN e protocolo.
  - **Inventário de Placas de Rede & Endereços MAC:** Tabela com status ativo/inativo, tipo (Ethernet/Wi-Fi/Virtual), MAC físico, IPv4 configurado e indicação de placa primária (rota padrão).
- **Gráficos Empilhados Verticais:**
  - Gráfico de Throughput (Mbps).
  - Gráfico de Latência (ms).
  - Gráfico dedicado de **Histórico de Perda de Pacotes (%)** com escala de 0% a 100%.

---

## 2. Ações Operacionais em Tempo Real

No cabeçalho da visão isolada, o operador conta com:
- **Botão "Atualizar" / Auto-refresh:** Recarrega os dados instantaneamente com animação visual de rotação no ícone.
- **Botão "Forçar Ping":** Executa um teste de ICMP instantâneo a partir da VM em direção ao nó, abrindo um modal com visualizador estilo console/terminal.
- **Botão "Detalhes do PC":** Abre modal com histórico aprofundado de memória, processador e sistema operacional da máquina.
- **Botão "Ver Máquinas do Switch":** Filtra a tabela geral para exibir todas as sondas que estão plugadas no mesmo switch corporativo identificado pelo LLDP.

