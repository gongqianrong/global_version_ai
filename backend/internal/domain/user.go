package domain

import "time"

// User represents a registered user.
type User struct {
	ID              int64     `json:"id"`
	Email           string    `json:"email"`
	Nickname        string    `json:"nickname"`
	PasswordHash    string    `json:"-"`
	GlobalAccountID string    `json:"global_account_id"` // 国际版用户唯一ID
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
