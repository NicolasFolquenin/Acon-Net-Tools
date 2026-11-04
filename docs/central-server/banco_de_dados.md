# Banco de Dados & Modelagem — MongoDB

O Servidor Central utiliza o **MongoDB** como repositório principal de dados.  
Para garantir compatibilidade ampla com ambientes virtualizados e hypervisors mais antigos (que não suportam instruções AVX do processador), a imagem oficial recomendada é a **MongoDB 4.4**.

---

## 1. Coleções Principais

### `metrics` (Séries Temporais / Telemetria)
Armazena cada pacote de medição enviado pelas sondas.
- **Campos Principais:**
  - `node_id` (string): Identificador único da sonda (geralmente o hostname ou UUID).
  - `timestamp` (ISODate): Data e hora da conclusão do teste.
  - `client_version` (string): Versão do software da sonda (ex: `"2.4.1"`).
  - `group_name` (string): Grupo ao qual a sonda pertencia no momento da medição.
  - `metrics` (object):
    - `download_mbps` (float64)
    - `upload_mbps` (float64)
    - `ping_gateway` (float64)
    - `packet_loss` (float64)
    - `modular_results` (object): Mapa de resultados dos módulos dinâmicos.
  - `system_info` (object): Snapshot do hardware, adaptadores de rede, Wi-Fi e IP.
  - `switch_info` (object): Descoberta LLDP do switch corporativo, porta e VLAN.

### `nodes` (Registro Consolidado de Sondas)
Mantém o cadastro e o estado mais recente de cada nó monitorado.
- **Campos:**
  - `node_id` (string, PK)
  - `location` (string)
  - `group_id` / `group_name` (string)
  - `last_seen` (ISODate)
  - `is_online` (boolean)
  - `client_version` (string)
  - `last_metrics` (object)
  - `system_info` & `switch_info` (object)

### `profiles` (Perfis de Medição)
Define baterias de testes customizadas que podem ser atribuídas a grupos ou máquinas individuais.
- **Campos:**
  - `id` (string)
  - `name` (string)
  - `version` (string)
  - `is_default` (boolean)
  - `tests` (array de objetos de configuração de cada teste):
    - `id`, `type`, `enabled`, `target`, `interval_seconds`, `timeout_ms`, `count`.

### `groups` (Grupos de Máquinas)
Agrupamento lógico para distribuição em massa de perfis de teste.

---

## 2. Índices de Alta Performance

Para permitir consultas instantâneas no Dashboard NOC sem lentidão conforme o histórico cresce, os seguintes índices são criados automaticamente na inicialização da aplicação Go:

```javascript
// Busca rápida por máquina ordenada temporalmente
db.metrics.createIndex({ "node_id": 1, "timestamp": -1 });

// Agregação geral por janela de tempo
db.metrics.createIndex({ "timestamp": -1 });

// Consulta por versão do cliente (para identificar máquinas desatualizadas)
db.metrics.createIndex({ "client_version": 1 });

// Consulta de nós por última conexão
db.nodes.createIndex({ "last_seen": -1 });
```

---

## 3. Retenção e Políticas de Limpeza (TTL)

Por padrão, as métricas brutas podem ser configuradas com índices TTL (`expireAfterSeconds`) no MongoDB para expiração automática após 90 ou 180 dias, evitando consumo excessivo de disco em ambientes de grande porte com centenas de sondas.

