package models

import "time"

// User представляет пользователя в системе
type User struct {
	ID        int64     `json:"id"`
	GoogleID  string    `json:"google_id"`  // Google sub (subject)
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Picture   string    `json:"picture"`    // URL аватара
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

