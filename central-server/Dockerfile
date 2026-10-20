# ==========================================
# Estágio 1: Compilação em Go (Debian Bookworm)
# ==========================================
FROM golang:1.22-bookworm AS builder

WORKDIR /app

# Copia arquivos de dependência
COPY go.mod go.sum* ./
RUN go mod download

# Copia o código fonte
COPY . .

# Garante resolução completa dos checksums dos módulos
RUN go mod tidy

# Compila o binário estático para arquitetura Linux
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o central-server main.go

# ==========================================
# Estágio 2: Imagem Final Leve (Debian Slim)
# ==========================================
FROM debian:bookworm-slim

# Instala certificados SSL e curl para healthcheck
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    iputils-ping \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Cria usuário não-root por segurança
RUN useradd -u 1001 -r -s /bin/false appuser

# Copia o executável, frontend estático e configurações padrão
COPY --from=builder /app/central-server /app/central-server
COPY --from=builder /app/public /app/public
COPY --from=builder /app/config /app/config

# Ajusta permissões
RUN mkdir -p /app/config/machines && chown -R appuser:appuser /app

USER appuser

# Variáveis de ambiente padrão
ENV PORT=8080 \
    MONGO_URI=mongodb://mongo:27017 \
    MONGO_DB=network_central \
    MONGO_COLLECTION=metrics \
    API_TOKEN=super-secret-probe-token-2026 \
    STATIC_DIR=/app/public

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/central-server"]
