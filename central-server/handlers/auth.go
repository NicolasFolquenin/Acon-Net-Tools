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
	"regexp"
	"strings"
	"time"

	"central-server/config"
	"central-server/db"
	"central-server/models"

	"golang.org/x/crypto/bcrypt"
)

type contextKey string

const userContextKey contextKey = "authUser"
const sessionCookieName = "noc_session"

// UserSessionPayload armazena os dados do usuário autenticado no token
type UserSessionPayload struct {
	Username  string    `json:"username"`
	Role      string    `json:"role"` // "admin" ou "viewer" / "operator"
	ExpiresAt time.Time `json:"expires_at"`
}

// AuthHandler gerencia login, logout, verificação de perfil e gestão de usuários
type AuthHandler struct {
	cfg      *config.AppConfig
	dbClient *db.MongoDBClient
}

// NewAuthHandler cria uma nova instância de AuthHandler
func NewAuthHandler(cfg *config.AppConfig, dbClient ...*db.MongoDBClient) *AuthHandler {
	h := &AuthHandler{cfg: cfg}
	if len(dbClient) > 0 {
		h.dbClient = dbClient[0]
	}
	return h
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
	// 1. Verifica contas mestras do ambiente (.env)
	if req.Username == a.cfg.AdminUser && req.Password == a.cfg.AdminPass {
		role = "admin"
	} else if req.Username == a.cfg.ViewerUser && req.Password == a.cfg.ViewerPass {
		role = "viewer"
	} else if a.dbClient != nil {
		// 2. Verifica se existe na base de dados de usuários do MongoDB
		dbUser, err := a.dbClient.GetUserByUsername(r.Context(), strings.ToLower(req.Username))
		if err == nil && dbUser != nil {
			if errBcrypt := bcrypt.CompareHashAndPassword([]byte(dbUser.PasswordHash), []byte(req.Password)); errBcrypt == nil {
				role = dbUser.Role
			}
		}
	}

	if role == "" {
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

// HandleUsers processa GET (listar), POST (criar), PUT (alterar papel) e DELETE (remover) usuários
func (a *AuthHandler) HandleUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		a.listUsers(w, r)
	case http.MethodPost:
		a.createUser(w, r)
	case http.MethodPut, http.MethodPatch:
		a.updateUserRole(w, r)
	case http.MethodDelete:
		a.deleteUser(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "metodo nao permitido"})
	}
}

func (a *AuthHandler) listUsers(w http.ResponseWriter, r *http.Request) {
	var userList []models.UserResponse

	// 1. Adiciona usuários padrão do sistema (.env)
	if a.cfg.AdminUser != "" {
		userList = append(userList, models.UserResponse{
			Username:  a.cfg.AdminUser,
			Role:      "admin",
			AuthType:  "system",
			CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			IsSystem:  true,
		})
	}
	if a.cfg.ViewerUser != "" {
		userList = append(userList, models.UserResponse{
			Username:  a.cfg.ViewerUser,
			Role:      "operator",
			AuthType:  "system",
			CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			IsSystem:  true,
		})
	}

	// 2. Adiciona usuários criados no MongoDB (Locais e Keycloak)
	if a.dbClient != nil {
		dbUsers, err := a.dbClient.ListUsers(r.Context())
		if err == nil {
			for _, u := range dbUsers {
				authType := u.AuthType
				if authType == "" {
					authType = "local"
				}
				userList = append(userList, models.UserResponse{
					Username:  u.Username,
					Role:      u.Role,
					AuthType:  authType,
					Email:     u.Email,
					FullName:  u.FullName,
					CreatedAt: u.CreatedAt,
					LastLogin: u.LastLogin,
					IsSystem:  false,
				})
			}
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"users":  userList,
	})
}

func (a *AuthHandler) createUser(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "corpo da requisicao invalido"})
		return
	}

	req.Username = strings.TrimSpace(strings.ToLower(req.Username))
	req.Password = strings.TrimSpace(req.Password)
	authType := strings.ToLower(strings.TrimSpace(req.AuthType))
	if authType != "keycloak" {
		authType = "local"
	}

	if len(req.Username) < 3 || len(req.Username) > 32 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "O nome de usuario deve ter entre 3 e 32 caracteres"})
		return
	}

	validUsername := regexp.MustCompile(`^[a-z0-9_\.\-]+$`)
	if !validUsername.MatchString(req.Username) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Nome de usuario contem caracteres invalidos. Use apenas letras minusculas, numeros, ponto, hifen ou sublinhado."})
		return
	}

	// Senha é obrigatória apenas para contas locais
	var passwordHash string
	if authType == "local" {
		if len(req.Password) < 4 {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "A senha deve ter pelo menos 4 caracteres"})
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "falha ao processar senha"})
			return
		}
		passwordHash = string(hash)
	}

	// Verifica se conflita com contas do sistema
	if req.Username == strings.ToLower(a.cfg.AdminUser) || req.Username == strings.ToLower(a.cfg.ViewerUser) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Este nome de usuario e reservado pelo sistema"})
		return
	}

	if a.dbClient == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "banco de dados nao conectado"})
		return
	}

	// Verifica se já existe no banco
	existing, _ := a.dbClient.GetUserByUsername(r.Context(), req.Username)
	if existing != nil {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Ja existe um usuario cadastrado com este nome"})
		return
	}

	role := "operator"
	if req.IsAdmin {
		role = "admin"
	}

	curUser, _ := r.Context().Value(userContextKey).(*UserSessionPayload)
	createdBy := "admin"
	if curUser != nil {
		createdBy = curUser.Username
	}

	newUser := &models.User{
		Username:     req.Username,
		PasswordHash: passwordHash,
		Role:         role,
		AuthType:     authType,
		CreatedAt:    time.Now().UTC(),
		CreatedBy:    createdBy,
	}

	if err := a.dbClient.CreateUser(r.Context(), newUser); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "falha ao salvar usuario no banco de dados"})
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Usuario criado com sucesso",
		"user": models.UserResponse{
			Username:  newUser.Username,
			Role:      newUser.Role,
			AuthType:  newUser.AuthType,
			CreatedAt: newUser.CreatedAt,
			IsSystem:  false,
		},
	})
}

func (a *AuthHandler) updateUserRole(w http.ResponseWriter, r *http.Request) {
	if a.dbClient == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "banco de dados nao conectado"})
		return
	}

	var req models.UpdateUserRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "corpo da requisicao invalido"})
		return
	}

	req.Username = strings.TrimSpace(strings.ToLower(req.Username))
	req.Role = strings.TrimSpace(strings.ToLower(req.Role))

	if req.Role != "admin" && req.Role != "operator" && req.Role != "viewer" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Papel invalido. Escolha 'admin' ou 'operator'."})
		return
	}
	if req.Role == "viewer" {
		req.Role = "operator"
	}

	if req.Username == strings.ToLower(a.cfg.AdminUser) || req.Username == strings.ToLower(a.cfg.ViewerUser) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Usuarios padrao do sistema nao podem ter seu perfil alterado"})
		return
	}

	curUser, _ := r.Context().Value(userContextKey).(*UserSessionPayload)
	if curUser != nil && strings.ToLower(curUser.Username) == req.Username && req.Role != "admin" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Voce nao pode revogar sua propria permissao de administrador"})
		return
	}

	if err := a.dbClient.UpdateUserRole(r.Context(), req.Username, req.Role); err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Permissao do usuario '%s' atualizada para '%s'", req.Username, req.Role),
	})
}

func (a *AuthHandler) deleteUser(w http.ResponseWriter, r *http.Request) {
	if a.dbClient == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "banco de dados nao conectado"})
		return
	}

	target := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("username")))
	if target == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "parametro 'username' e obrigatorio"})
		return
	}

	if target == strings.ToLower(a.cfg.AdminUser) || target == strings.ToLower(a.cfg.ViewerUser) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Usuarios padrao do sistema nao podem ser removidos"})
		return
	}

	curUser, _ := r.Context().Value(userContextKey).(*UserSessionPayload)
	if curUser != nil && strings.ToLower(curUser.Username) == target {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Voce nao pode excluir o usuario que esta atualmente conectado"})
		return
	}

	if err := a.dbClient.DeleteUser(r.Context(), target); err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Usuario '%s' excluido com sucesso", target),
	})
}
