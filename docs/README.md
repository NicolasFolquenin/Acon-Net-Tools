# Índice Geral da Documentação — UEPG Net Tools

Bem-vindo à documentação oficial do ecossistema **UEPG Net Tools (Anel de Conectividade ACON / NTI)**.

A documentação principal está organizada em **3 Grandes Guias Mestres**, estruturados exatamente para atender às necessidades de visão geral, operação prática e engenharia de software:

---

## 📚 1. Os Três Guias Principais

| Documento | Descrição e Propósito |
| :--- | :--- |
| 📘 **[1. O Que É o Net-Tools e O Que Ele Faz](01_o_que_e_e_o_que_faz.md)** | Visão geral, objetivos institucionais, problemas que resolve no ACON/UEPG, métricas de rede (latência, perda, throughput, Wi-Fi, DNS), descoberta física de switches LLDP/CDP, inventário de placas, MACs e usuários. |
| 🚀 **[2. Como Utilizar o Net-Tools](02_como_utilizar.md)** | Manual prático de operação: como navegar no Dashboard Central, filtrar por grupo/VLAN, isolar máquinas, forçar pings, ver vizinhos de switch, instalar a sonda no Windows (`testar.bat`, firewall) e Linux (`install-probe.sh`, monitor CLI), e gerenciar a VM com `./deploy.sh`. |
| 🛠️ **[3. Documentação de Como Funciona o Código](03_como_funciona_o_codigo.md)** | Engenharia e arquitetura interna: fluxo ponta a ponta, ciclo de polling, executores modulares Go, captura de pacotes raw LLDP, buffer local SQLite, mitigação de falsos positivos de Wi-Fi, rotas REST, TimeSeries no MongoDB e frontend em JavaScript puro. |

---

## 🗂️ 2. Módulos Técnicos Complementares de Referência

Para aprofundamento em tópicos específicos de baixo nível, consulte as seções dedicadas:

### 🏛️ [Arquitetura do Sistema](arquitetura/visao_geral.md)
- [Visão Geral da Arquitetura](arquitetura/visao_geral.md): Topologia do sistema distribuído e requisitos de infraestrutura.
- [Fluxo de Telemetria e Sincronização](arquitetura/fluxo_telemetria.md): Do disparo do teste na sonda até a persistência no banco e agregação de dados.
- [Segurança, Autenticação e Permissões](arquitetura/seguranca_e_rede.md): Tokens de autenticação, JWT e níveis de acesso.

### 🐧 [Servidor Central (`central-server`)](central-server/api_referencia.md)
- [Referência da REST API](central-server/api_referencia.md): Documentação completa dos endpoints de telemetria, nós, perfis, grupos e downloads.
- [Banco de Dados & Schema](central-server/banco_de_dados.md): Modelagem no MongoDB TimeSeries, coleções e índices de performance.
- [Dashboard Web NOC](central-server/dashboard_web.md): Interface de monitoramento em tempo real, gráficos Chart.js e diagnósticos.

### 🖥️ [Sonda Cliente (`client-probe`)](client-probe/ciclo_de_execucao.md)
- [Ciclo de Execução e Agendamento](client-probe/ciclo_de_execucao.md): Operação daemon/background e buffer local SQLite.
- [Módulos de Teste e Medição](client-probe/modulos_de_teste.md): Detalhamento dos executores: Latência ICMP, Perda, Throughput, DNS e LLDP.
- [Sistema de Auto-Update e Versionamento](client-probe/auto_update.md): Versionamento semântico (`v2.4.1`), validação de hash SHA256 e hot-restart.
- [Modo Desktop Gráfico (Wails v2)](client-probe/modo_desktop_wails.md): Arquitetura do aplicativo desktop Windows.

### 🚀 [Deploy e Operações](deploy/servidor_docker.md)
- [Deploy do Servidor Central (Docker & VM)](deploy/servidor_docker.md): Instalação em 1 comando com `deploy.sh`.
- [Procedimento de Atualização das Sondas](deploy/procedimento_atualizacao_sondas.md): Guia de rollout de novas versões de sonda.
- [Instalação da Sonda em Linux](deploy/sonda_linux.md): Script de instalação rápida e serviço `systemd`.
- [Instalação e Empacotamento Windows](deploy/sonda_windows.md): Geração de executáveis e testes em lote.

### 🎨 [UX/UI & Design System](ux-ui/diretrizes_visuais.md)
- [Diretrizes Visuais e Componentes](ux-ui/diretrizes_visuais.md): Sistema de cores institucional UEPG, tipografia e badges.
- [Dashboard & Gráficos Interativos](ux-ui/dashboard_e_graficos.md): Padronização do Chart.js e responsividade.
