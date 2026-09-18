# Central Server Coletor de Telemetria de Rede (Linux/Debian)

Servidor central coletor de alto desempenho escrito em **Go 1.22+**, integrado ao **MongoDB** com coleções **Time Series** nativas e **Dashboard Web** responsivo com **Chart.js**.

---

## 🏛️ Arquitetura

- **Backend:** Go 1.22+ com driver oficial `go.mongodb.org/mongo-driver/v2/mongo`.
- **Database:** MongoDB 7.0+ com coleção Time Series (`network_central.metrics`):
  - `timeField`: `timestamp`
  - `metaField`: `metadata` (`node_id`, `location`)
  - `granularity`: `hours`
- **Dashboard Web:** Servido diretamente em `GET /` a partir de `./public`.
- **Autenticação:** Header HTTP `Authorization: Bearer <api_token>`.

---

## 🚀 Como Executar no Debian Linux

### Opção 1: Via Docker Compose (Recomendado)

1. Clone ou transfira a pasta `/central-server` para a máquina Debian:
   ```bash
   cd central-server
   ```
2. Inicie os containers do MongoDB e do Go Server:
   ```bash
   docker compose up -d --build
   ```
3. Verifique os logs:
   ```bash
   docker compose logs -f central-server
   ```
4. Acesse o Dashboard Web no navegador:
   `http://<IP_DA_VM_DEBIAN>:8080`

---

### Opção 2: Instalação Nativa (Systemd + MongoDB)

1. **Instalar MongoDB no Debian 12 (Bookworm):**
   ```bash
   sudo apt-get update && sudo apt-get install -y gnupg curl
   curl -fsSL https://www.mongodb.org/static/pgp/server-7.0.asc | \
      sudo gpg -o /usr/share/keyrings/mongodb-server-7.0.gpg --dearmor
   echo "deb [ signed-by=/usr/share/keyrings/mongodb-server-7.0.gpg ] http://repo.mongodb.org/apt/debian bookworm/mongodb-org/7.0 main" | \
      sudo tee /etc/apt/sources.list.d/mongodb-org-7.0.list
   sudo apt-get update && sudo apt-get install -y mongodb-org
   sudo systemctl enable --now mongod
   ```

2. **Compilar e Instalar o Servidor Central:**
   ```bash
   cd central-server
   go build -o central-server main.go
   sudo mkdir -p /opt/central-server
   sudo cp central-server /opt/central-server/
   sudo cp -r public /opt/central-server/
   ```

3. **Configurar o Serviço Systemd:**
   ```bash
   sudo cp systemd/network-monitor.service /etc/systemd/system/
   sudo systemctl daemon-reload
   sudo systemctl enable --now network-monitor
   sudo systemctl status network-monitor
   ```

---

## 📡 Testando os Endpoints via cURL

### 1. Ingestão de Métricas (`POST /api/v1/metrics`)
```bash
curl -X POST http://localhost:8080/api/v1/metrics \
  -H "Authorization: Bearer super-secret-probe-token-2026" \
  -H "Content-Type: application/json" \
  -d '{
    "node_id": "minipc-dc-01",
    "location": "Guarapuava-DC1",
    "timestamp": "2026-09-11T15:00:00Z",
    "download_mbps": 850.5,
    "upload_mbps": 720.0,
    "ping_gateway": 1.2
  }'
```
**Resposta (HTTP 201 Created):**
```json
{
  "message": "Metrica armazenada com sucesso",
  "status": "success",
  "timestamp": "2026-09-11T15:00:00Z"
}
```

### 2. Consulta de Séries do Dashboard (`GET /api/v1/dashboard/metrics`)
```bash
curl -s http://localhost:8080/api/v1/dashboard/metrics | jq .
```
Ou filtrar por um nó específico:
```bash
curl -s "http://localhost:8080/api/v1/dashboard/metrics?node_id=minipc-dc-01" | jq .
```

### 3. Autenticação do Dashboard (`POST /api/v1/auth/login`)
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "admin123"}'
```

### 4. Exclusão de Computador por Admin (`DELETE /api/v1/nodes`)
```bash
curl -X DELETE "http://localhost:8080/api/v1/nodes?node_id=minipc-dc-01" \
  --cookie "noc_session=<TOKEN_ADMIN>"
```

---

## 🔍 Inspecionando o MongoDB Time Series

Acesse o shell do MongoDB (`mongosh`):
```bash
mongosh
```
Comandos:
```javascript
use network_central;

// Verificar metadados da coleção Time Series
db.getCollectionInfos({ name: "metrics" });

// Consultar as últimas 5 medições
db.metrics.find().sort({ timestamp: -1 }).limit(5);

// Agregação de média de Download por nó
db.metrics.aggregate([
  {
    $group: {
      _id: "$metadata.node_id",
      avg_download: { $avg: "$download_mbps" },
      avg_upload: { $avg: "$upload_mbps" },
      avg_ping: { $avg: "$ping_gateway" }
    }
  }
]);
```
