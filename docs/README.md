# Índice Geral da Documentação Técnica — UEPG Net Tools

Bem-vindo à documentação modular da plataforma **UEPG Net Tools (Anel de Conectividade ACON)**.  
O sistema é estruturado em duas camadas principais: o **Servidor Central Coletor** (backend Go com MongoDB Time Series e Dashboard NOC) e as **Sondas Clientes Multi-plataforma** (Go CLI / Wails v2 para Linux e Windows).

Esta documentação está particionada em módulos temáticos para facilitar o desenvolvimento, manutenção e operação.

---

## Estrutura da Documentação

### 1. [Arquitetura do Sistema](arquitetura/visao_geral.md)
- [Visão Geral da Arquitetura](arquitetura/visao_geral.md): Topologia do sistema distribuído, tecnologias envolvidas e requisitos de infraestrutura.
- [Fluxo de Telemetria e Sincronização](arquitetura/fluxo_telemetria.md): Do disparo do teste na sonda até a persistência no banco e agregação de dados.
- [Segurança, Autenticação e Permissões](arquitetura/seguranca_e_rede.md): Tokens de autenticação, JWT, integração SSO com Keycloak/OIDC e níveis de acesso (`admin` vs `operador`).

### 2. [Servidor Central (`central-server`)](central-server/api_referencia.md)
- [Referência da REST API](central-server/api_referencia.md): Documentação completa dos endpoints de telemetria, nós, perfis, grupos e download de versões.
- [Banco de Dados & Schema](central-server/banco_de_dados.md): Modelagem no MongoDB, coleções temporais, retenção de dados e índices de performance.
- [Dashboard Web NOC](central-server/dashboard_web.md): Interface de monitoramento em tempo real, visualização agregada e isolada por máquina, badges de versão e diagnósticos.

### 3. [Sonda Cliente (`client-probe`)](client-probe/ciclo_de_execucao.md)
- [Ciclo de Execução e Agendamento](client-probe/ciclo_de_execucao.md): Operação daemon/background, sincronização com slots centrais (`:00` e `:30`) e buffer local SQLite.
- [Módulos de Teste e Medição](client-probe/modulos_de_teste.md): Detalhamento dos executores: Latência ICMP, Perda de Pacotes, Throughput HTTP, DNS e Descoberta LLDP/CDP.
- [Sistema de Auto-Update e Versionamento](client-probe/auto_update.md): Como funciona o versionamento semântico (`v2.4.1`), consulta de hash/versão e troca atômica de binários em disco.
- [Modo Desktop Gráfico (Wails v2)](client-probe/modo_desktop_wails.md): Arquitetura do aplicativo desktop Windows, IPC Go/JavaScript, temas Dark/Light e empacotamento em `dist/`.

### 4. [Deploy e Operações](deploy/servidor_docker.md)
- [Deploy do Servidor Central (Docker & VM)](deploy/servidor_docker.md): Instalação em 1 comando com `deploy.sh`, variáveis de ambiente `.env` e persistência.
- [Instalação da Sonda em Linux](deploy/sonda_linux.md): Script de instalação rápida, serviço `systemd` e monitor interativo via terminal (`uepg-probe-monitor`).
- [Instalação e Empacotamento Windows](deploy/sonda_windows.md): Geração de executáveis, instalador gráfico e integração com bandeja do sistema.

### 5. [UX/UI & Design System](ux-ui/diretrizes_visuais.md)
- [Diretrizes Visuais e Componentes](ux-ui/diretrizes_visuais.md): Sistema de cores dual (Dark/Light), tipografia Inter/JetBrains Mono, badges semânticos e consistência visual.
- [Dashboard & Gráficos Interativos](ux-ui/dashboard_e_graficos.md): Padronização do Chart.js, empilhamento vertical de gráficos, tratamento de séries temporais e responsividade.

