# Ciclo de Execução e Agendamento — Sonda Cliente

A sonda cliente (`client-probe`) opera de forma totalmente autônoma em segundo plano, executando rotinas de medição de rede periódicas e mantendo resiliência contra falhas.

---

## 1. Modos de Agendamento

A sonda suporta dois modos de agendamento:

### 1.1. Modo Sincronizado Centralmente (Padrão Corporativo)
Quando conectada ao Servidor Central (`server_url` configurada e válida):
- Todas as sondas da instituição disparam suas baterias de teste rigorosamente nos minutos `:00` e `:30` de cada hora.
- Isso permite analisar a fotografia do estado da rede universitária em instantes temporais uniformes e correlacionar eventos.
- A função `getNextSyncSlot(time.Now())` calcula com exatidão o próximo slot de meia hora, aguardando o despertar do temporizador Go.

### 1.2. Modo Independente / Intervalar
Quando em rede desconectada do servidor ou configurada localmente:
- A sonda executa a cada `interval_minutes` configurado no `config.json` (por exemplo, a cada 60 minutos).

---

## 2. Inicialização e Bootstrap

Ao iniciar (seja via serviço `systemd` no Linux ou executável Wails no Windows), a sonda executa a seguinte sequência:

1. **Limpeza de Binários Antigos:** A função `CleanupOldBinaries()` remove resíduos de patches anteriores (`.old` e `.new`).
2. **Checagem de Atualização Assíncrona:** Consulta o servidor central para verificar se há uma versão mais nova disponível no repositório.
3. **Inicialização do SQLite:** Conecta ao banco `probe_local.db` e executa migrações de tabela caso necessário.
4. **Coleta de Hardware e Interfaces:** Detecta modelo do processador, total de RAM, hostname, adaptadores de rede físicos, MACs e IPs locais.
5. **Escuta de Anúncio LLDP:** Inicia thread de escuta passiva de pacotes LLDP/CDP na interface de rede para identificar o switch e porta física.
6. **Sincronização de Perfil Remoto:** Consulta `/api/v1/profiles/host` informando o hostname e MAC. Caso o servidor retorne um perfil customizado, a sonda atualiza seu plano de testes.
7. **Disparo do Agendador em Segundo Plano:** Registra o próximo slot de execução e fica em espera.

