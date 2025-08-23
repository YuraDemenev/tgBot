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

func (t *TaskServer) SendTask(ctx context.Context, req *taskpb.SendTaskRequest) (*taskpb.SendTaskResponse, error) {
	logrus.Infof("user: %s, send task", req.UserName)
	res := &taskpb.SendTaskResponse{}

	errUserMessage, statusGRPC, err := t.repo.SaveTask(req)
	if err != nil {
		logrus.Errorf("Can`t save task, user: %s", req.UserName)
		res.Status = statusGRPC
		res.UserErrorMessage = errUserMessage
		return res, err
	}
	res.Status = statusGRPC
	res.UserErrorMessage = errUserMessage
	return res, nil
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

func (t *TaskServer) DeleteTask(ctx context.Context, req *taskpb.DeleteTaskRequest) (*taskpb.DeleteTaskResponse, error) {
	logrus.Info("user %s, start delete task%s", req.UserName, req.TaskNumber)
	err := t.repo.DeleteTask(req.UserName, int(req.TaskNumber))
	if err != nil {
		logrus.Errorf("user%s can`t delete task, err:%v", req.UserName, err)
		res := &taskpb.DeleteTaskResponse{
			Ok:     false,
			Status: &status.Status{Code: int32(codes.Internal)},
		}
		return res, err
	}

	res := &taskpb.DeleteTaskResponse{
		Ok:     true,
		Status: &status.Status{Code: int32(codes.OK)},
	}
	return res, nil
}

func (t *TaskServer) ChangeTask(ctx context.Context, req *taskpb.ChangeTaskRequest) (*taskpb.ChangeTaskResponse, error) {
	logrus.Info("user %s, start change task%s", req.UserName, req.TaskNum)
	err := t.repo.ChangeTask(req)
	if err != nil {
		logrus.Errorf("user%s can`t change task, err:%v", req.UserName, err)
		res := &taskpb.ChangeTaskResponse{
			Ok:     false,
			Status: &status.Status{Code: int32(codes.Internal)},
		}
		return res, err
	}

	res := &taskpb.ChangeTaskResponse{
		Ok:     true,
		Status: &status.Status{Code: int32(codes.OK)},
	}
	return res, nil
}
