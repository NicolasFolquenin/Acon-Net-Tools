package models

import "time"

// User representa um usuário cadastrado no sistema
type User struct {
	ID           string    `bson:"_id,omitempty" json:"id,omitempty"`
	Username     string    `bson:"username" json:"username"`
	PasswordHash string    `bson:"password_hash" json:"-"`
	Role         string    `bson:"role" json:"role"` // "admin" ou "operator"
	CreatedAt    time.Time `bson:"created_at" json:"created_at"`
	CreatedBy    string    `bson:"created_by,omitempty" json:"created_by,omitempty"`
}

// CreateUserRequest corpo da requisição POST /api/v1/users
type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	IsAdmin  bool   `json:"is_admin"`
}

// UserResponse representação pública de um usuário
type UserResponse struct {
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	IsSystem  bool      `json:"is_system"`
}
