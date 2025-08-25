package repositories

import (
	"tgbot/bot-service/protoGenFiles/tgBot/bot-service/protoGenFiles/taskpb"
	"tgbot/task-service/internal/cache"
	"tgbot/task-service/internal/rabbitmq"

	"github.com/jmoiron/sqlx"
	"google.golang.org/genproto/googleapis/rpc/status"
)

type Tasks interface {
	SaveTask(req *taskpb.SendTaskRequest) (string, *status.Status, error)
	GetTasks(req *taskpb.GetTasksRequest) (string, *status.Status, []taskpb.Task, error)
	DeleteTask(userName string, taskNum int) (string, *status.Status, error)
	ChangeTask(req *taskpb.ChangeTaskRequest) (string, *status.Status, error)
}

type Repository struct {
	Tasks
}

func NewRepository(db *sqlx.DB, cache cache.Cache, r *rabbitmq.RabbitMQ) *Repository {
	return &Repository{
		Tasks: NewTasksPostgres(db, cache, r),
	}
}
