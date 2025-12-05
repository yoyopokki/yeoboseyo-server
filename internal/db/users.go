package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yeoboseyo/server/internal/models"
)

// GetOrCreateUser находит пользователя по google_id или создаёт нового
// Возвращает пользователя и флаг, указывающий, был ли он создан (true) или найден (false)
func GetOrCreateUser(ctx context.Context, pool *pgxpool.Pool, googleID, email, name, picture string) (*models.User, bool, error) {
	// Сначала пытаемся найти существующего пользователя
	user, err := GetUserByGoogleID(ctx, pool, googleID)
	if err == nil && user != nil {
		// Пользователь найден, обновляем информацию (на случай, если изменились email, name, picture)
		updatedUser, err := UpdateUser(ctx, pool, user.ID, email, name, picture)
		if err != nil {
			return user, false, nil // Возвращаем старую версию, если обновление не удалось
		}
		return updatedUser, false, nil
	}

	// Пользователь не найден, создаём нового
	newUser, err := CreateUser(ctx, pool, googleID, email, name, picture)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create user: %w", err)
	}

	return newUser, true, nil
}

// GetUserByGoogleID находит пользователя по Google ID
func GetUserByGoogleID(ctx context.Context, pool *pgxpool.Pool, googleID string) (*models.User, error) {
	var user models.User
	query := `
		SELECT id, google_id, email, name, picture, created_at, updated_at
		FROM users
		WHERE google_id = $1
	`

	err := pool.QueryRow(ctx, query, googleID).Scan(
		&user.ID,
		&user.GoogleID,
		&user.Email,
		&user.Name,
		&user.Picture,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Пользователь не найден
		}
		return nil, fmt.Errorf("failed to get user by google_id: %w", err)
	}

	return &user, nil
}

// GetUserByID находит пользователя по ID
func GetUserByID(ctx context.Context, pool *pgxpool.Pool, userID int64) (*models.User, error) {
	var user models.User
	query := `
		SELECT id, google_id, email, name, picture, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	err := pool.QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.GoogleID,
		&user.Email,
		&user.Name,
		&user.Picture,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return &user, nil
}

// CreateUser создаёт нового пользователя
func CreateUser(ctx context.Context, pool *pgxpool.Pool, googleID, email, name, picture string) (*models.User, error) {
	var user models.User
	query := `
		INSERT INTO users (google_id, email, name, picture, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, google_id, email, name, picture, created_at, updated_at
	`

	err := pool.QueryRow(ctx, query, googleID, email, name, picture).Scan(
		&user.ID,
		&user.GoogleID,
		&user.Email,
		&user.Name,
		&user.Picture,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

// UpdateUser обновляет информацию о пользователе
func UpdateUser(ctx context.Context, pool *pgxpool.Pool, userID int64, email, name, picture string) (*models.User, error) {
	var user models.User
	query := `
		UPDATE users
		SET email = $2, name = $3, picture = $4, updated_at = NOW()
		WHERE id = $1
		RETURNING id, google_id, email, name, picture, created_at, updated_at
	`

	err := pool.QueryRow(ctx, query, userID, email, name, picture).Scan(
		&user.ID,
		&user.GoogleID,
		&user.Email,
		&user.Name,
		&user.Picture,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return &user, nil
}

