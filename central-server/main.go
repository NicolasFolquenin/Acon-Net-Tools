package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
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
	authHandler := handlers.NewAuthHandler(cfg, mongoClient)
	oidcHandler := handlers.NewOIDCHandler(cfg, authHandler, mongoClient)
	profileHandler := handlers.NewProfileHandler(cfg)
	machineProfileHandler := handlers.NewMachineProfileHandler(cfg, mongoClient)
	groupsHandler := handlers.NewGroupsHandler(mongoClient, cfg)
	modulesHandler := handlers.NewModulesHandler(cfg, mongoClient)

	mux := http.NewServeMux()

	// Rota estática GET / serve a pasta ./public com headers no-cache para assets dinâmicos
	fileServer := http.FileServer(http.Dir(cfg.StaticDir))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" || strings.HasSuffix(path, ".html") || strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".css") {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
		}
		fileServer.ServeHTTP(w, r)
	})

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"healthy","service":"central-server","time":"` + time.Now().UTC().Format(time.RFC3339) + `"}`))
	})

	// Speedtest endpoints para medição real das sondas
	mux.HandleFunc("/api/v1/speedtest/download", handlers.SpeedtestDownloadHandler)
	mux.HandleFunc("/api/v1/speedtest/upload", handlers.SpeedtestUploadHandler)

	// Endpoints de Autenticação (Login, Logout, Sessão Atual)
	mux.HandleFunc("/api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("/api/v1/auth/logout", authHandler.Logout)
	mux.HandleFunc("/api/v1/auth/me", authHandler.Me)

	// Endpoints de Autenticação Institucional (OpenID Connect / Keycloak)
	mux.HandleFunc("/api/v1/auth/oidc/config", oidcHandler.ConfigHandler)
	mux.HandleFunc("/api/v1/auth/oidc/login", oidcHandler.LoginHandler)
	mux.HandleFunc("/api/v1/auth/oidc/callback", oidcHandler.CallbackHandler)

	// Endpoints de Gerenciamento de Usuários (Apenas Administrador)
	mux.HandleFunc("/api/v1/users", authHandler.RequireAdmin(authHandler.HandleUsers))

	// Endpoints de Perfil Dinâmico de Medição das Sondas
	// 1. GET /api/v1/probes/profile/{hostname} -> perfil individual por máquina
	// 2. GET /api/v1/probes/profile -> perfil legado global
	// 3. PUT /api/v1/probes/profile -> atualização do perfil global pelo admin
	mux.HandleFunc("/api/v1/probes/profile/", machineProfileHandler.HandleGetProfileByHostname)
	mux.HandleFunc("/api/v1/probes/profile", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			authHandler.RequireAdmin(profileHandler.UpdateProfile)(w, r)
			return
		}
		profileHandler.GetProfile(w, r)
	})

	// Auto-registro de novas sondas (quando 404 no perfil)
	mux.HandleFunc("/api/v1/probes/register", handlers.RequireBearerToken(cfg.APIToken, machineProfileHandler.HandleRegisterProbe))

	// Endpoints Administrativos para Máquinas, Grupos e Módulos
	mux.HandleFunc("/api/v1/admin/machines", authHandler.RequireAdmin(machineProfileHandler.HandleAdminMachines))
	mux.HandleFunc("/api/v1/admin/machines/", authHandler.RequireAdmin(machineProfileHandler.HandleAdminMachines))
	mux.HandleFunc("/api/v1/admin/groups", authHandler.RequireAdmin(machineProfileHandler.HandleAdminGroups))
	mux.HandleFunc("/api/v1/admin/groups/", authHandler.RequireAdmin(machineProfileHandler.HandleAdminGroups))
	mux.HandleFunc("/api/v1/admin/modules", authHandler.RequireAdmin(modulesHandler.HandleModules))
	mux.HandleFunc("/api/v1/admin/modules/", authHandler.RequireAdmin(modulesHandler.HandleModules))

	// Endpoints de Grupos de Testes e VLANs (compatibilidade com dashboard e sondas legadas)
	mux.HandleFunc("/api/v1/config/probe", handlers.RequireBearerToken(cfg.APIToken, groupsHandler.GetProbeConfig))
	mux.HandleFunc("/api/v1/groups", authHandler.RequireAdmin(groupsHandler.HandleGroups))
	mux.HandleFunc("/api/v1/groups/", authHandler.RequireAdmin(groupsHandler.HandleGroups))

	// Endpoints da API oficial
	// POST /api/v1/metrics protegido por Bearer Token
	mux.HandleFunc("/api/v1/metrics", handlers.RequireBearerToken(cfg.APIToken, metricsHandler.PostMetric))

	// GET /api/v1/dashboard/metrics protegido por Autenticação (Admin ou Operador)
	mux.HandleFunc("/api/v1/dashboard/metrics", authHandler.RequireAuth(metricsHandler.GetDashboardMetrics))

	// DELETE /api/v1/nodes protegido exclusivamente para perfil Administrador
	mux.HandleFunc("/api/v1/nodes", authHandler.RequireAdmin(metricsHandler.DeleteNode))

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
