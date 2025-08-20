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
		return &status.Status{Code: int32(codes.Internal)}, err
	}

	return &status.Status{Code: int32(codes.OK)}, nil
}

func (t *TaskServer) GetTasks(ctx context.Context, req *taskpb.GetTasksRequest) (*taskpb.GetTasksResponse, error) {
	logrus.Infof("user: %s, getting tasks", req.UserName)
	tasks, err := t.repo.GetTasks(req)
	if err != nil {
		logrus.Errorf("Can`t get tasks, user: %s", req.UserName)
		res := &taskpb.GetTasksResponse{
			Ok:   false,
			Task: nil,
		}
		return res, err
	}

	respTasks := make([]*taskpb.Task, 0, len(tasks))
	for i := range tasks {
		respTasks = append(respTasks, &tasks[i])
	}

	res := &taskpb.GetTasksResponse{
		Ok:   true,
		Task: respTasks,
	}
	return res, nil
}
