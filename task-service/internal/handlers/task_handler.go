package handlers

import (
	"context"
	"tgbot/bot-service/protoGenFiles/tgBot/bot-service/protoGenFiles/taskpb"
	"tgbot/task-service/internal/repositories"

	"github.com/sirupsen/logrus"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
)

type TaskServer struct {
	taskpb.UnimplementedTaskServiceServer
	repo *repositories.Repository
}

func NewTaskServer(repo *repositories.Repository) *TaskServer {
	return &TaskServer{
		repo: repo,
	}
}

func (t *TaskServer) SendTask(ctx context.Context, req *taskpb.SendTaskRequest) (*status.Status, error) {
	logrus.Infof("user: %s, send task", req.UserName)
	err := t.repo.SaveTask(req)
	if err != nil {
		logrus.Errorf("Can`t save task, user: %s", req.UserName)
		return &status.Status{codes.Internal}, err
	}
	//TODO change status
	return &status.Status{}, nil
}
