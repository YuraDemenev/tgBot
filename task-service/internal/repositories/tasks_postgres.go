package repositories

import (
	"tgbot/task-service/internal/cache"

	"github.com/jmoiron/sqlx"
)

type TasksPostgres struct {
	db          *sqlx.DB
	cacheClient cache.Cache
}

func NewTasksPostgres(db *sqlx.DB, cache cache.Cache) Tasks {
	return &TasksPostgres{db: db, cacheClient: cache}
}
