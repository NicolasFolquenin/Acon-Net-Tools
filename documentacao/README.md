# 📚 Documentação Técnica - UEPG Net Tools & Polling App Acon

Bem-vindo à documentação oficial do ecossistema de monitoramento de rede e telemetria distribuída desenvolvido para a **Universidade Estadual de Ponta Grossa (UEPG)** no projeto **ACON**.

---

## 🧭 Sumário da Documentação

Esta pasta contém guias detalhados sobre a arquitetura, novos módulos de rede, auto-atualização de sondas e procedimentos operacionais:

| Documento | Descrição |
| :--- | :--- |
| **[1. Arquitetura do Sistema](arquitetura_do_sistema.md)** | Visão completa da arquitetura distribuída: Central Server, Client Probe, MongoDB, SQLite e fluxos de dados. |
| **[2. Módulos de Rede: Wi-Fi, MACs e Multi-DNS](novos_modulos_wifi_macs_dns.md)** | Detalhamento técnico da medição de sinal Wi-Fi (dBm/%), inventário multi-placa/MACs e benchmark de servidores DNS. |
| **[3. Sistema de Auto-Update e Patches](sistema_de_auto_update.md)** | Mecanismo de auto-patching transparente das sondas nas máquinas clientes, endpoints de versão e substituição segura de binários. |
| **[4. Guia de Compilação e Deploy](guia_de_compilacao_e_deploy.md)** | Passo a passo para compilar o Central Server, gerar executáveis e instaladores Windows/Linux, e implantar novas versões. |

---

## 🌟 Principais Recursos do Sistema

1. **Monitoramento Ativo e Telemetria em Tempo Real**:
   - Medição de Throughput (Download e Upload real com chunks dinâmicos).
   - Latência e Perda de Pacotes (ICMP Ping nativo).
   - Descoberta automática de Topologia Switch e Porta via **LLDP** (Link Layer Discovery Protocol).
2. **Novos Módulos Especializados**:
   - **Multi-Interface & Múltiplos MACs**: Mapeia todas as placas do host, elegendo a interface com a rota padrão ativa.
   - **Telemetria Wi-Fi**: Qualidade do sinal sem fio (dBm e %), taxas TX/RX, SSID, BSSID e canal.
   - **Benchmark Multi-DNS**: Medição comparativa de latência resolvendo contra DNS Institucional UEPG, Cloudflare e Google.
3. **Gestão Centralizada**:
   - Dashboard NOC em tempo real com visão consolidada e isolada de cada máquina.
   - Perfis dinâmicos de teste por máquina e grupos de VLANs sem reiniciar clientes.
   - **Auto-Update Nativo**: Sondas clientes atualizam seu próprio executável em segundo plano ao detectar nova versão no servidor.
