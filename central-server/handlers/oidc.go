package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"central-server/config"
	"central-server/db"
	"central-server/models"
)

const oidcStateCookieName = "noc_oidc_state"

// OIDCDiscoveryDocument armazena os endpoints descobertos do Identity Provider
type OIDCDiscoveryDocument struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
	EndSessionEndpoint    string `json:"end_session_endpoint"`
}

// OIDCHandler gerencia o fluxo de autenticação OpenID Connect (Keycloak / SSO)
type OIDCHandler struct {
	cfg        *config.AppConfig
	auth       *AuthHandler
	dbClient   *db.MongoDBClient
	httpClient *http.Client

	mu        sync.RWMutex
	discovery *OIDCDiscoveryDocument
}

// NewOIDCHandler instancia um novo handler OIDC
func NewOIDCHandler(cfg *config.AppConfig, auth *AuthHandler, dbClient *db.MongoDBClient) *OIDCHandler {
	return &OIDCHandler{
		cfg:      cfg,
		auth:     auth,
		dbClient: dbClient,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// ConfigHandler expõe se o OIDC está ativo para o frontend exibir o botão de SSO
func (o *OIDCHandler) ConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"enabled":     o.cfg.OIDCEnabled,
		"issuer_url":  o.cfg.OIDCIssuerURL,
		"client_id":   o.cfg.OIDCClientID,
		"button_text": "Entrar com Keycloak",
	})
}

// LoginHandler inicia o fluxo OIDC redirecionando o navegador para o Keycloak
func (o *OIDCHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if !o.cfg.OIDCEnabled {
		http.Error(w, `{"error": "autenticacao OIDC nao esta habilitada no servidor"}`, http.StatusBadRequest)
		return
	}

	disc, err := o.getDiscovery(r.Context())
	if err != nil {
		log.Printf("[OIDC][ERRO] Falha ao obter discovery document do Keycloak: %v", err)
		http.Error(w, `{"error": "falha de comunicacao com o servidor de autenticacao institucional"}`, http.StatusBadGateway)
		return
	}

	// Gera state aleatório criptográfico para proteção contra CSRF
	stateBytes := make([]byte, 24)
	if _, err := rand.Read(stateBytes); err != nil {
		http.Error(w, `{"error": "falha interna de geracao de estado"}`, http.StatusInternalServerError)
		return
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)

	// Salva state em cookie temporário HttpOnly (10 minutos)
	http.SetCookie(w, &http.Cookie{
		Name:     oidcStateCookieName,
		Value:    state,
		Path:     "/",
		Expires:  time.Now().UTC().Add(10 * time.Minute),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// Redireciona para o authorization_endpoint do Keycloak
	authURL, err := url.Parse(disc.AuthorizationEndpoint)
	if err != nil {
		http.Error(w, `{"error": "url de autorizacao invalida"}`, http.StatusInternalServerError)
		return
	}

	q := authURL.Query()
	q.Set("client_id", o.cfg.OIDCClientID)
	q.Set("response_type", "code")
	q.Set("scope", "openid profile email groups")
	q.Set("redirect_uri", o.cfg.OIDCRedirectURL)
	q.Set("state", state)
	authURL.RawQuery = q.Encode()

	http.Redirect(w, r, authURL.String(), http.StatusFound)
}

// CallbackHandler recebe o authorization code do Keycloak, troca por tokens e inicia a sessão
func (o *OIDCHandler) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	if !o.cfg.OIDCEnabled {
		http.Error(w, "OIDC desabilitado", http.StatusBadRequest)
		return
	}

	// 1. Valida se o Keycloak retornou erro
	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		errDesc := r.URL.Query().Get("error_description")
		log.Printf("[OIDC] Erro retornado pelo Keycloak: %s (%s)", errMsg, errDesc)
		http.Redirect(w, r, "/?auth_error="+url.QueryEscape(errMsg), http.StatusFound)
		return
	}

	// 2. Valida o State para prevenir CSRF
	queryState := r.URL.Query().Get("state")
	stateCookie, err := r.Cookie(oidcStateCookieName)
	if err != nil || stateCookie.Value == "" || stateCookie.Value != queryState {
		log.Printf("[OIDC][AVISO] State invalido ou ausente: cookie=%v, query=%s", err, queryState)
		http.Redirect(w, r, "/?auth_error=state_invalido", http.StatusFound)
		return
	}

	// Limpa cookie de state
	http.SetCookie(w, &http.Cookie{
		Name:     oidcStateCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Redirect(w, r, "/?auth_error=codigo_ausente", http.StatusFound)
		return
	}

	disc, err := o.getDiscovery(r.Context())
	if err != nil {
		log.Printf("[OIDC][ERRO] Falha ao obter endpoints OIDC: %v", err)
		http.Redirect(w, r, "/?auth_error=falha_servidor_auth", http.StatusFound)
		return
	}

	// 3. Troca Authorization Code por Access Token
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", o.cfg.OIDCRedirectURL)
	data.Set("client_id", o.cfg.OIDCClientID)
	data.Set("client_secret", o.cfg.OIDCClientSecret)

	reqToken, err := http.NewRequestWithContext(r.Context(), http.MethodPost, disc.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		http.Redirect(w, r, "/?auth_error=erro_interno", http.StatusFound)
		return
	}
	reqToken.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	respToken, err := o.httpClient.Do(reqToken)
	if err != nil || respToken.StatusCode != http.StatusOK {
		log.Printf("[OIDC][ERRO] Falha na troca do code por token: err=%v, status=%d", err, respToken.StatusCode)
		http.Redirect(w, r, "/?auth_error=falha_token", http.StatusFound)
		return
	}
	defer respToken.Body.Close()

	var tokenRes struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
	}
	if err := json.NewDecoder(respToken.Body).Decode(&tokenRes); err != nil || tokenRes.AccessToken == "" {
		http.Redirect(w, r, "/?auth_error=token_invalido", http.StatusFound)
		return
	}

	// 4. Obtém perfil do usuário no UserInfo Endpoint
	reqUserinfo, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, disc.UserinfoEndpoint, nil)
	reqUserinfo.Header.Set("Authorization", "Bearer "+tokenRes.AccessToken)

	respUserinfo, err := o.httpClient.Do(reqUserinfo)
	if err != nil || respUserinfo.StatusCode != http.StatusOK {
		log.Printf("[OIDC][ERRO] Falha ao consultar UserInfo: err=%v, status=%d", err, respUserinfo.StatusCode)
		http.Redirect(w, r, "/?auth_error=falha_userinfo", http.StatusFound)
		return
	}
	defer respUserinfo.Body.Close()

	var rawUserInfo map[string]interface{}
	if err := json.NewDecoder(respUserinfo.Body).Decode(&rawUserInfo); err != nil {
		http.Redirect(w, r, "/?auth_error=perfil_invalido", http.StatusFound)
		return
	}

	// Extrai nome de usuário institucional
	username := extractUsername(rawUserInfo)
	if username == "" {
		log.Printf("[OIDC][ERRO] Nenhum identificador valido de usuario encontrado no payload: %+v", rawUserInfo)
		http.Redirect(w, r, "/?auth_error=usuario_nao_identificado", http.StatusFound)
		return
	}

	email, _ := rawUserInfo["email"].(string)
	name, _ := rawUserInfo["name"].(string)

	// 5. Determina Role padrão inicial baseado nos grupos do Keycloak
	defaultRole := "operator"
	if o.hasAdminGroup(rawUserInfo) {
		defaultRole = "admin"
	}

	// 6. Sincroniza com MongoDB e obtém a Role definitiva (concede soberania ao Admin do Net Tools!)
	finalRole := defaultRole
	if o.dbClient != nil {
		userModel := &models.User{
			Username:  username,
			Role:      defaultRole,
			AuthType:  "keycloak",
			Email:     email,
			FullName:  name,
			LastLogin: time.Now().UTC(),
		}
		savedUser, errUpsert := o.dbClient.UpsertKeycloakUser(r.Context(), userModel)
		if errUpsert == nil && savedUser != nil {
			finalRole = savedUser.Role
		}
	}

	// 7. Gera sessão autenticada padrão do Net Tools
	payload := UserSessionPayload{
		Username:  username,
		Role:      finalRole,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}

	tokenStr, err := o.auth.generateSignedToken(payload)
	if err != nil {
		log.Printf("[OIDC][ERRO] Falha ao assinar sessao para %s: %v", username, err)
		http.Redirect(w, r, "/?auth_error=falha_sessao", http.StatusFound)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    tokenStr,
		Path:     "/",
		Expires:  payload.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	log.Printf("[OIDC] Login institucional realizado com sucesso: usuario=%s, role=%s", username, finalRole)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (o *OIDCHandler) getDiscovery(ctx context.Context) (*OIDCDiscoveryDocument, error) {
	o.mu.RLock()
	if o.discovery != nil {
		defer o.mu.RUnlock()
		return o.discovery, nil
	}
	o.mu.RUnlock()

	o.mu.Lock()
	defer o.mu.Unlock()

	// Checa novamente caso outra goroutine tenha preenchido
	if o.discovery != nil {
		return o.discovery, nil
	}

	wellKnownURL := o.cfg.OIDCIssuerURL + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wellKnownURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status inesperado %d de %s: %s", resp.StatusCode, wellKnownURL, string(body))
	}

	var disc OIDCDiscoveryDocument
	if err := json.NewDecoder(resp.Body).Decode(&disc); err != nil {
		return nil, fmt.Errorf("falha ao decodificar openid-configuration: %w", err)
	}

	o.discovery = &disc
	log.Printf("[OIDC] Discovery OpenID carregado com sucesso de: %s", wellKnownURL)
	return o.discovery, nil
}

func extractUsername(userInfo map[string]interface{}) string {
	if u, ok := userInfo["preferred_username"].(string); ok && strings.TrimSpace(u) != "" {
		return strings.ToLower(strings.TrimSpace(u))
	}
	if e, ok := userInfo["email"].(string); ok && strings.TrimSpace(e) != "" {
		parts := strings.Split(e, "@")
		return strings.ToLower(strings.TrimSpace(parts[0]))
	}
	if s, ok := userInfo["sub"].(string); ok && strings.TrimSpace(s) != "" {
		return strings.ToLower(strings.TrimSpace(s))
	}
	return ""
}

func (o *OIDCHandler) hasAdminGroup(userInfo map[string]interface{}) bool {
	targetGroup := strings.ToLower(strings.TrimSpace(o.cfg.OIDCAdminGroup))
	if targetGroup == "" {
		return false
	}

	// 1. Verifica claim "groups" (padrão Keycloak / OIDC)
	if rawGroups, ok := userInfo["groups"]; ok {
		if groupList, isList := rawGroups.([]interface{}); isList {
			for _, g := range groupList {
				strGroup := strings.ToLower(fmt.Sprintf("%v", g))
				strGroup = strings.TrimPrefix(strGroup, "/")
				if strGroup == targetGroup {
					return true
				}
			}
		}
	}

	// 2. Verifica claim "roles"
	if rawRoles, ok := userInfo["roles"]; ok {
		if roleList, isList := rawRoles.([]interface{}); isList {
			for _, r := range roleList {
				if strings.ToLower(fmt.Sprintf("%v", r)) == targetGroup {
					return true
				}
			}
		}
	}

	// 3. Verifica realm_access.roles
	if realmAccess, ok := userInfo["realm_access"].(map[string]interface{}); ok {
		if rawRoles, ok := realmAccess["roles"].([]interface{}); ok {
			for _, r := range rawRoles {
				if strings.ToLower(fmt.Sprintf("%v", r)) == targetGroup {
					return true
				}
			}
		}
	}

	return false
}
