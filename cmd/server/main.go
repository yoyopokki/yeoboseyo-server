package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"

	"github.com/yeoboseyo/server/internal/db"
	"github.com/yeoboseyo/server/internal/httpapi"
)

func main() {
	_ = godotenv.Load()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// Инициализируем подключение к PostgreSQL перед запуском HTTP-сервера.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx)
	if err != nil {
		zlog.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer pool.Close()

	// Выполняем миграции
	if err := db.RunMigrations(ctx, pool); err != nil {
		zlog.Fatal().Err(err).Msg("failed to run migrations")
	}

	// Прокидываем пул в httpapi-пакет для использования в хендлерах.
	httpapi.SetDB(pool)

	r := mux.NewRouter()

	httpapi.RegisterRoutes(r)

	// CORS: разрешаем запросы с фронтенда на http://localhost:3000
	r.Use(corsMiddleware)

	addr := ":" + getEnv("PORT", "8080")
	zlog.Info().Msgf("starting server on %s", addr)

	if err := http.ListenAndServe(addr, r); err != nil {
		zlog.Fatal().Err(err).Msg("server failed")
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// corsMiddleware добавляет CORS-заголовки и обрабатывает preflight OPTIONS-запросы.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		acrMethod := r.Header.Get("Access-Control-Request-Method")
		acrHeaders := r.Header.Get("Access-Control-Request-Headers")

		zlog.Info().
			Str("path", r.URL.Path).
			Str("method", r.Method).
			Str("origin", origin).
			Str("acr_method", acrMethod).
			Str("acr_headers", acrHeaders).
			Msg("CORS middleware hit")

		if origin == "http://localhost:3000" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if acrHeaders != "" {
			// Разрешаем все запрошенные заголовки preflight-запроса
			w.Header().Set("Access-Control-Allow-Headers", acrHeaders+", Content-Type, Authorization")
		} else {
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Vary", "Origin")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}




