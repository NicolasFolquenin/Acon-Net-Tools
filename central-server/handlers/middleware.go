package handlers

import (
	"log"
	"net/http"
	"strings"
	"time"
)

// ResponseWriterInterceptor captura o status code HTTP para logging
type ResponseWriterInterceptor struct {
	http.ResponseWriter
	StatusCode int
}

func (w *ResponseWriterInterceptor) WriteHeader(code int) {
	w.StatusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// LoggerMiddleware gera log estruturado para cada requisição HTTP
func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w *ResponseWriterInterceptorOrStandard, r *http.Request) {
		start := time.Now()
		rw := &ResponseWriterInterceptor{ResponseWriter: w, StatusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		elapsed := time.Since(start)
		log.Printf("[HTTP] %s %s - %d (%v)", r.Method, r.URL.Path, rw.StatusCode, elapsed)
	})
}

// ResponseWriterInterceptorOrStandard interface wrapper
type ResponseWriterInterceptorOrStandard = http.ResponseWriter

// CORSMiddleware permite requisições cross-origin para a API REST
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept, X-Requested-With")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireBearerToken valida o cabeçalho Authorization: Bearer <token>
func RequireBearerToken(expectedToken string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error": "cabecalho Authorization ausente"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, `{"error": "formato de token invalido, use 'Bearer <token>'"}`, http.StatusUnauthorized)
			return
		}

		token := strings.TrimSpace(parts[1])
		if token != expectedToken {
			log.Printf("[AUTH][AVISO] Tentativa de acesso com token invalido a partir de %s", r.RemoteAddr)
			http.Error(w, `{"error": "token nao autorizado"}`, http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}
