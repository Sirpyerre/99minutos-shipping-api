package domain

import "time"

const (
	RoleAdmin  = "admin"
	RoleClient = "client"
)

// User models an authenticated actor in the system.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email,omitempty"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	ClientID     string    `json:"client_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
