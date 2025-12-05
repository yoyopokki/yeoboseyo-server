package httpapi

import "github.com/jackc/pgx/v5/pgxpool"

// dbPool — пакетный уровень, чтобы хендлеры могли использовать БД.
// По мере роста проекта лучше заменить на явную передачу зависимостей (структура Server и т.п.).
var dbPool *pgxpool.Pool

func SetDB(pool *pgxpool.Pool) {
	dbPool = pool
}

// DB отдаёт текущий пул (может быть nil, если SetDB не вызывали).
func DB() *pgxpool.Pool {
	return dbPool
}


