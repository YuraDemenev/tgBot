package repositories

import (
	"tgbot/task-service/internal/cache"

	"github.com/jmoiron/sqlx"
)

type Tasks interface {
}

type Repository struct {
	Tasks
}

func NewRepository(db *sqlx.DB, cache cache.Cache) *Repository {
	return &Repository{
		Tasks: NewTasksPostgres(db, cache),
	}
}
