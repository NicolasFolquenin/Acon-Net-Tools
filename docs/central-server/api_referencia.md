# Referência da REST API — Servidor Central

A API do Servidor Central é versionada sob o prefixo `/api/v1`.  
Todas as respostas são em formato JSON (`Content-Type: application/json; charset=utf-8`).

---

## 1. Métricas & Telemetria

### `POST /api/v1/metrics`
Registra uma nova medição enviada por uma sonda cliente.
- **Headers:** `Authorization: Bearer <token>`
- **Body:** Objeto `MetricPayloadInput` com métricas consolidadas, hardware, versão e switch.
- **Respostas:**
  - `201 Created`: Métrica gravada com sucesso. Retorna ID do documento inserido.
  - `400 Bad Request`: JSON malformado ou campos obrigatórios ausentes (`node_id`).
  - `401 Unauthorized`: Token de autorização ausente ou inválido.

### `GET /api/v1/metrics/node/:id`
Consulta o histórico temporal de métricas de uma sonda específica.
- **Parâmetros de Query:**
  - `period`: Intervalo de consulta (ex: `1h`, `6h`, `24h`, `7d`, `30d`). Padrão: `24h`.
  - `limit`: Quantidade máxima de registros retornados (padrão: 50, máx: 500).
- **Resposta:**
  ```json
  {
    "node_id": "cpdv0412",
    "total_records": 24,
    "records": [ ... ]
  }
  ```

### `GET /api/v1/metrics/latest`
Retorna a leitura mais recente de todas as sondas ativas (usado para alimentar os cards do Dashboard geral).

---

## 2. Gestão de Nós e Diagnósticos

### `GET /api/v1/nodes`
Lista todas as sondas conhecidas pelo servidor, com status online/offline, IP, localização, versão do cliente e switch conectado.

### `GET /api/v1/nodes/:id/diagnostic`
Retorna o diagnóstico aprofundado de um nó específico, incluindo dados de hardware, Wi-Fi, inventário de placas de rede e versão da sonda (`client_version`).

### `POST /api/v1/nodes/:id/ping`
Dispara um teste de Ping sob demanda a partir do servidor em direção ao nó selecionado. Retorna saída em tempo real e tempo de resposta.

### `DELETE /api/v1/nodes/:id`
Remove o registro e históricos de uma máquina inativa (requer permissão `admin`).

---

## 3. Perfis e Grupos de Medição

### `GET /api/v1/profiles`
Lista todos os perfis de medição cadastrados no servidor.

### `GET /api/v1/profiles/host`
Chamado pela sonda cliente no momento da sincronização para saber quais testes deve executar.
- **Parâmetros de Query:**
  - `hostname`: Nome da máquina cliente.
  - `mac`: Endereço MAC físico da interface primária.
  - `group`: Grupo sugerido localmente.
- **Resposta:** JSON com a lista de testes ativos (`tests`), frequências e alvos.

### `POST /api/v1/profiles` / `PUT /api/v1/profiles/:id`
Cria ou atualiza um perfil de testes (requer privilégio de administrador).

### `GET /api/v1/groups` / `POST /api/v1/groups`
Gerenciamento de grupos lógicos de sondas (ex: "Bootstrap", "Grupo ACON", "Laboratórios").

---

## 4. Distribuição de Binários & Auto-Update

### `GET /api/v1/download/version`
Retorna a versão mais recente publicada para as sondas e os metadados dos binários disponíveis para download.
- **Resposta Exemplo:**
  ```json
  {
    "version": "2.4.1",
    "published_at": "2026-11-03T07:00:00Z",
    "platforms": {
      "windows-amd64": {
        "url": "/downloads/probe-windows-amd64.exe",
        "available": true,
        "size_bytes": 14285824
      },
      "linux-amd64": {
        "url": "/downloads/probe-linux-amd64",
        "available": true,
        "size_bytes": 11845120
      }
    }
  }
  ```

### `GET /downloads/:filename`
Entrega o binário executável ou instalador para atualização em disco.

