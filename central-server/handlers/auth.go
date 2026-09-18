package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"central-server/config"
)

type contextKey string

const userContextKey contextKey = "authUser"
const sessionCookieName = "noc_session"

// UserSessionPayload armazena os dados do usuário autenticado no token
type UserSessionPayload struct {
	Username  string    `json:"username"`
	Role      string    `json:"role"` // "admin" ou "viewer"
	ExpiresAt time.Time `json:"expires_at"`
}

// AuthHandler gerencia login, logout e verificação de perfil
type AuthHandler struct {
	cfg *config.AppConfig
}

// NewAuthHandler cria uma nova instância de AuthHandler
func NewAuthHandler(cfg *config.AppConfig) *AuthHandler {
	return &AuthHandler{cfg: cfg}
}

// LoginRequest corpo da requisição POST /api/v1/auth/login
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login valida as credenciais e emite o token assinado
func (a *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "metodo nao permitido"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "corpo da requisicao invalido"})
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Password = strings.TrimSpace(req.Password)

	var role string
	if req.Username == a.cfg.AdminUser && req.Password == a.cfg.AdminPass {
		role = "admin"
	} else if req.Username == a.cfg.ViewerUser && req.Password == a.cfg.ViewerPass {
		role = "viewer"
	} else {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Usuario ou senha incorretos"})
		return
	}

	// Gera sessão com validade de 24 horas
	payload := UserSessionPayload{
		Username:  req.Username,
		Role:      role,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}

	token, err := a.generateSignedToken(payload)
	if err != nil {
		log.Printf("[AUTH][ERRO] Falha ao assinar token: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "falha interna de autenticacao"})
		return
	}

	// Define cookie seguro
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  payload.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	log.Printf("[AUTH] Login realizado com sucesso: usuario=%s, perfil=%s", req.Username, role)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"token":  token,
		"user": map[string]string{
			"username": req.Username,
			"role":     role,
		},
	})
}

// Logout encerra a sessão limpando o cookie
func (a *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Sessao encerrada com sucesso",
	})
}

// Me retorna o usuário atualmente autenticado
func (a *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	user := a.getUserFromRequest(r)
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": false,
			"error":         "Nao autenticado",
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"authenticated": true,
		"user": map[string]string{
			"username": user.Username,
			"role":     user.Role,
		},
	})
}

// RequireAuth valida se a requisição possui sessão ativa
func (a *AuthHandler) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := a.getUserFromRequest(r)
		if user == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "Autenticacao necessaria. Faca login para continuar.",
			})
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next(w, r.WithContext(ctx))
	}
}

// RequireAdmin valida se o usuário possui papel de administrador
func (a *AuthHandler) RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return a.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		user, _ := r.Context().Value(userContextKey).(*UserSessionPayload)
		if user == nil || user.Role != "admin" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "Acesso negado. Apenas administradores podem executar esta acao.",
			})
			return
		}

		next(w, r)
	})
}

func (a *AuthHandler) getUserFromRequest(r *http.Request) *UserSessionPayload {
	// 1. Tenta ler do cookie
	var tokenStr string
	if cookie, err := r.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
		tokenStr = cookie.Value
	}

	// 2. Se nao houver cookie, tenta o header Authorization
	if tokenStr == "" {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if tokenStr == "" {
		return nil
	}

	payload, err := a.verifySignedToken(tokenStr)
	if err != nil {
		return nil
	}

	if time.Now().UTC().After(payload.ExpiresAt) {
		return nil
	}

	return payload
}

func (a *AuthHandler) generateSignedToken(payload UserSessionPayload) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	payloadB64 := base64.RawURLEncoding.EncodeToString(data)
	sig := a.computeHmac(payloadB64)

	return fmt.Sprintf("%s.%s", payloadB64, sig), nil
}

func (a *AuthHandler) verifySignedToken(token string) (*UserSessionPayload, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("formato de token invalido")
	}

	payloadB64 := parts[0]
	sig := parts[1]

	expectedSig := a.computeHmac(payloadB64)
	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return nil, fmt.Errorf("assinatura de token invalida")
	}

	data, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return nil, fmt.Errorf("falha ao decodificar payload")
	}

	var payload UserSessionPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("falha no unmarshal do token")
	}

	return &payload, nil
}

func (a *AuthHandler) computeHmac(message string) string {
	mac := hmac.New(sha256.New, []byte(a.cfg.SessionSecret))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}
