# Segurança, Autenticação e Permissões

Este documento aborda os mecanismos de segurança aplicados no tráfego de dados, autenticação de sondas e controle de acesso ao Dashboard Web.

---

## 1. Comunicação Segura e Isolamento

- **Transporte Criptografado (TLS/HTTPS):** Toda comunicação entre as sondas e o Servidor Central deve ocorrer via HTTPS, evitando interceptação de credenciais e parâmetros de rede física (endereços MAC, switches e VLANs).
- **Isolamento de Sondas:** As sondas não abrem portas públicas de escuta no host. Toda a comunicação é puramente **outbound** (a sonda inicia a requisição HTTP contra o servidor central).
- **Execução com Privilégios Mínimos:**
  - No Linux, comandos que exigem raw sockets (como `ping` nativo ou escuta de LLDP) utilizam as capabilities Linux do kernel (`cap_net_raw+ep`) em vez de exigir execução contínua como superusuário root irrestrito.
  - No Windows, as chamadas de ICMP utilizam a API nativa ou o binário padrão `ping.exe` sem demandar elevação de UAC constante na interface Wails.

---

## 2. Autenticação das Sondas (Bearer Token)

Cada sonda cliente se autentica no Servidor Central através de um Bearer Token configurado no arquivo `config.json` ou variável de ambiente:

```http
POST /api/v1/metrics HTTP/1.1
Host: akvorado.acon.dev.uepg.br
Authorization: Bearer <API_TOKEN>
Content-Type: application/json
```

O Servidor Central valida o token contra a chave mestre definida em `config.json` ou no `.env` (`API_TOKEN`). Caso o token seja inválido, a requisição é rejeitada com `HTTP 401 Unauthorized`.

---

## 3. Autenticação de Usuários no Dashboard Web

O acesso ao Dashboard de monitoramento possui dois métodos de controle de acesso:

### 3.1. Autenticação Local (Nativa)
Ideal para redes isoladas ou quando não há provedor de identidade externo:

| Usuário | Senha Padrão | Nível de Acesso | Permissões |
| :--- | :--- | :--- | :--- |
| **`admin`** | `admin123` | Administrador Completo | Visualizar dashboards, criar/editar perfis e grupos, excluir máquinas inativas, disparar diagnósticos remotos e configurar tolerâncias. |
| **`operador`** | `operador123` | Leitura Operacional | Visualização de telemetria em tempo real, consulta de históricos e filtros. Ações destrutivas e de alteração de configuração são bloqueadas. |

### 3.2. Integração SSO Institucional (Keycloak / OpenID Connect)
O backend Go possui suporte nativo a OpenID Connect (OIDC). Quando configuradas as variáveis de ambiente `OIDC_ISSUER_URL`, `OIDC_CLIENT_ID` e `OIDC_CLIENT_SECRET`, o Dashboard redireciona os operadores para o Keycloak institucional da UEPG, mapeando os grupos e papéis institucionais.

