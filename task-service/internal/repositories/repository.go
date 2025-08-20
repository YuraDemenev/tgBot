package repositories

import (
	"tgbot/bot-service/protoGenFiles/tgBot/bot-service/protoGenFiles/taskpb"
	"tgbot/task-service/internal/cache"

	"github.com/jmoiron/sqlx"
)

type Tasks interface {
	SaveTask(req *taskpb.SendTaskRequest) error
	GetTasks(req *taskpb.GetTasksRequest) ([]taskpb.Task, error)
}

type Repository struct {
	Tasks
}

func NewRepository(db *sqlx.DB, cache cache.Cache) *Repository {
	return &Repository{
		Tasks: NewTasksPostgres(db, cache),
	}
}
