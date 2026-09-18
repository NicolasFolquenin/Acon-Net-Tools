package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"central-server/config"
)

func setupTestAuthConfig() *config.AppConfig {
	return &config.AppConfig{
		AdminUser:     "admin",
		AdminPass:     "admin123",
		ViewerUser:    "operador",
		ViewerPass:    "operador123",
		SessionSecret: "test-secret-12345",
	}
}

func TestAuthLoginAdmin(t *testing.T) {
	cfg := setupTestAuthConfig()
	auth := NewAuthHandler(cfg)

	body, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "admin123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	auth.Login(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("esperado status 200, recebido %d", resp.StatusCode)
	}

	cookies := resp.Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == sessionCookieName {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("esperado cookie de sessao, mas nenhum foi encontrado")
	}

	var resData map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&resData)
	userMap, ok := resData["user"].(map[string]interface{})
	if !ok || userMap["role"] != "admin" {
		t.Fatalf("esperado perfil 'admin', recebido %+v", userMap)
	}
}

func TestAuthLoginInvalidCredentials(t *testing.T) {
	cfg := setupTestAuthConfig()
	auth := NewAuthHandler(cfg)

	body, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "wrong-password",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	auth.Login(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("esperado status 401, recebido %d", resp.StatusCode)
	}
}

func TestAuthMe(t *testing.T) {
	cfg := setupTestAuthConfig()
	auth := NewAuthHandler(cfg)

	// 1. Tenta acessar sem cookie
	reqUnauth := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	wUnauth := httptest.NewRecorder()
	auth.Me(wUnauth, reqUnauth)
	if wUnauth.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("esperado 401 para usuario sem cookie, recebido %d", wUnauth.Result().StatusCode)
	}

	// 2. Faz login de operador e acessa com cookie
	loginBody, _ := json.Marshal(map[string]string{
		"username": "operador",
		"password": "operador123",
	})
	reqLogin := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	wLogin := httptest.NewRecorder()
	auth.Login(wLogin, reqLogin)

	cookies := wLogin.Result().Cookies()

	reqAuth := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	for _, c := range cookies {
		reqAuth.AddCookie(c)
	}
	wAuth := httptest.NewRecorder()
	auth.Me(wAuth, reqAuth)

	if wAuth.Result().StatusCode != http.StatusOK {
		t.Fatalf("esperado 200 com cookie valido, recebido %d", wAuth.Result().StatusCode)
	}
}

func TestRequireAdminPermission(t *testing.T) {
	cfg := setupTestAuthConfig()
	auth := NewAuthHandler(cfg)

	protectedHandler := auth.RequireAdmin(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Operador tenta acessar handler de admin -> 403 Forbidden
	tokenOp, _ := auth.generateSignedToken(UserSessionPayload{
		Username:  "operador",
		Role:      "viewer",
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	})
	reqOp := httptest.NewRequest(http.MethodDelete, "/api/v1/nodes?node_id=pc-01", nil)
	reqOp.AddCookie(&http.Cookie{Name: sessionCookieName, Value: tokenOp})
	wOp := httptest.NewRecorder()
	protectedHandler(wOp, reqOp)

	if wOp.Result().StatusCode != http.StatusForbidden {
		t.Fatalf("esperado 403 Forbidden para operador, recebido %d", wOp.Result().StatusCode)
	}

	// Admin acessa handler de admin -> 200 OK
	tokenAdmin, _ := auth.generateSignedToken(UserSessionPayload{
		Username:  "admin",
		Role:      "admin",
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	})
	reqAdmin := httptest.NewRequest(http.MethodDelete, "/api/v1/nodes?node_id=pc-01", nil)
	reqAdmin.AddCookie(&http.Cookie{Name: sessionCookieName, Value: tokenAdmin})
	wAdmin := httptest.NewRecorder()
	protectedHandler(wAdmin, reqAdmin)

	if wAdmin.Result().StatusCode != http.StatusOK {
		t.Fatalf("esperado 200 OK para admin, recebido %d", wAdmin.Result().StatusCode)
	}
}
