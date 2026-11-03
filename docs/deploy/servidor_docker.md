# Deploy do Servidor Central (Docker & VM)

O Servidor Central é projetado para instalação em um único comando em máquinas virtuais Linux (Debian 11/12, Ubuntu 20/22/24, Rocky Linux, CentOS).

---

## 1. Instalação em 1 Comando

```bash
# 1. Clonar o repositório oficial
git clone https://git.uepg.br/anelconectividade/polling-app-acon.git
cd polling-app-acon/central-server

# 2. Executar o script automatizado
chmod +x deploy.sh
sudo ./deploy.sh
```

---

## 2. O que o `deploy.sh` Executa

1. **Checagem de Dependências:** Instala automaticamente `docker` e o plugin `docker compose` oficial caso não existam.
2. **Assistente Interativo no Primeiro Deploy:** Pergunta no terminal da VM:
   - Porta HTTP do servidor (padrão: `8090`).
   - IP ou Domínio institucional (ex: `akvorado.acon.dev.uepg.br`).
   - Token mestre da API (`API_TOKEN`).
   - Senha do administrador (`ADMIN_PASSWORD`).
   Grava as respostas no arquivo `.env` persistente.
3. **Persistência de Dados:** O banco MongoDB e as configurações são mapeados para volumes locais do host:
   - `./mongo_data`: Dados do MongoDB Time Series.
   - `./config`: Arquivos de configuração de grupos e perfis padrão.
   - `.env`: Segredos e parâmetros de rede.
4. **Inicialização do Docker Compose:** Sobe os containers `central-server` e `central-mongo` em background.
5. **Healthcheck:** Aguarda a API responder com `HTTP 200` no endpoint `/health` e exibe o resumo de acesso.

---

## 3. Comandos de Gerenciamento

```bash
./deploy.sh             # Inicialização rápida (se já configurado, sobe direto)
./deploy.sh --wizard    # Reabre o assistente para alterar parâmetros
./deploy.sh --status    # Consulta a saúde dos containers e banco de dados
./deploy.sh --logs      # Acompanha logs em tempo real
./deploy.sh --down      # Interrompe containers mantendo os dados intactos
./deploy.sh -y          # Execução silenciosa (para pipelines de CI/CD)
```

