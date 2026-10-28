package models

import "time"

// User representa um usuário cadastrado no sistema (Local ou Keycloak/OIDC)
type User struct {
	ID           string    `bson:"_id,omitempty" json:"id,omitempty"`
	Username     string    `bson:"username" json:"username"`
	PasswordHash string    `bson:"password_hash,omitempty" json:"-"`
	Role         string    `bson:"role" json:"role"` // "admin" ou "operator"
	AuthType     string    `bson:"auth_type" json:"auth_type"` // "local" ou "keycloak"
	Email        string    `bson:"email,omitempty" json:"email,omitempty"`
	FullName     string    `bson:"full_name,omitempty" json:"full_name,omitempty"`
	CreatedAt    time.Time `bson:"created_at" json:"created_at"`
	LastLogin    time.Time `bson:"last_login,omitempty" json:"last_login,omitempty"`
	CreatedBy    string    `bson:"created_by,omitempty" json:"created_by,omitempty"`
}

// CreateUserRequest corpo da requisição POST /api/v1/users
type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
	IsAdmin  bool   `json:"is_admin"`
	AuthType string `json:"auth_type,omitempty"` // "local" (padrão) ou "keycloak"
}

// UpdateUserRoleRequest corpo da requisição PUT /api/v1/users/role
type UpdateUserRoleRequest struct {
	Username string `json:"username"`
	Role     string `json:"role"` // "admin" ou "operator"
}

// UserResponse representação pública de um usuário para listagem no painel
type UserResponse struct {
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	AuthType  string    `json:"auth_type"` // "local", "keycloak", "system"
	Email     string    `json:"email,omitempty"`
	FullName  string    `json:"full_name,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	LastLogin time.Time `json:"last_login,omitempty"`
	IsSystem  bool      `json:"is_system"`
}
