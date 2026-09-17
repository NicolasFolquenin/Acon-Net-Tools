package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"central-server/config"
	"central-server/db"
	"central-server/handlers"
)

func main() {
	log.Println("==========================================================")
	log.Println(" INICIANDO SERVIDOR CENTRAL DE TELEMETRIA DE REDE (GO)   ")
	log.Println("==========================================================")

	// 1. Carrega configurações do ambiente
	cfg := config.LoadConfig()

	// 2. Conecta ao MongoDB Time Series
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	mongoClient, err := db.InitMongoDB(ctx, cfg)
	if err != nil {
		log.Printf("[MAIN][AVISO] Falha ao conectar no MongoDB: %v. O servidor continuara em modo resiliente.", err)
	}
	defer func() {
		if mongoClient != nil {
			disconnectCtx, cancelDisc := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancelDisc()
			_ = mongoClient.Disconnect(disconnectCtx)
		}
	}()

	// 3. Inicializa Handlers
	metricsHandler := handlers.NewMetricsHandler(mongoClient, cfg)

	mux := http.NewServeMux()

	// Rota estática GET / serve a pasta ./public
	fileServer := http.FileServer(http.Dir(cfg.StaticDir))
	mux.Handle("/", fileServer)

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"healthy","service":"central-server","time":"` + time.Now().UTC().Format(time.RFC3339) + `"}`))
	})

	// Speedtest endpoints para medição real das sondas
	mux.HandleFunc("/api/v1/speedtest/download", handlers.SpeedtestDownloadHandler)
	mux.HandleFunc("/api/v1/speedtest/upload", handlers.SpeedtestUploadHandler)

	// Endpoints da API oficial
	// POST /api/v1/metrics protegido por Bearer Token
	mux.HandleFunc("/api/v1/metrics", handlers.RequireBearerToken(cfg.APIToken, metricsHandler.PostMetric))

	// GET /api/v1/dashboard/metrics
	mux.HandleFunc("/api/v1/dashboard/metrics", metricsHandler.GetDashboardMetrics)

	// POST /api/v1/seed (utilitário para gerar histórico de 24h para demonstração visual)
	mux.HandleFunc("/api/v1/seed", metricsHandler.SeedSampleData)

	// Envolve com middlewares de CORS e Logging
	rootHandler := handlers.CORSMiddleware(handlers.LoggerMiddleware(mux))

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      rootHandler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 4. Inicia servidor HTTP em goroutine
	go func() {
		log.Printf("[MAIN] Servidor Central escutando na porta %s (Dashboard: http://localhost:%s)", cfg.Port, cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[MAIN] Erro fatal no servidor HTTP: %v", err)
		}
	}()

	// 5. Graceful shutdown em caso de sinal do sistema (SIGINT / SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[MAIN] Sinal de interrupcao recebido. Encerrando servidor graciosamente...")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("[MAIN] Erro durante o shutdown do servidor: %v", err)
	}

	log.Println("[MAIN] Servidor finalizado com sucesso.")
}
