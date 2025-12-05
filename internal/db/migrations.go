package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// RunMigrations выполняет SQL миграции из директории migrations
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	migrationsDir := "migrations"
	
	// Проверяем существование директории
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		log.Warn().Str("dir", migrationsDir).Msg("migrations directory not found, skipping migrations")
		return nil
	}

	// Читаем все .sql файлы из директории migrations
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		migrationPath := filepath.Join(migrationsDir, entry.Name())
		log.Info().Str("file", migrationPath).Msg("running migration")

		sql, err := os.ReadFile(migrationPath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", migrationPath, err)
		}

		// Выполняем SQL
		_, err = pool.Exec(ctx, string(sql))
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", migrationPath, err)
		}

		log.Info().Str("file", migrationPath).Msg("migration completed successfully")
	}

	return nil
}

