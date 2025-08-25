package handlers

import (
	"context"
	"tgbot/bot-service/protoGenFiles/tgBot/bot-service/protoGenFiles/taskpb"
	"tgbot/task-service/internal/repositories"

	"github.com/sirupsen/logrus"
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
		statusGRPC.Message = err.Error()
		res.Status = statusGRPC
		res.UserErrorMessage = errUserMessage
		return res, nil
	}
	res.Status = statusGRPC
	res.UserErrorMessage = errUserMessage
	return res, nil
}

func (t *TaskServer) GetTasks(ctx context.Context, req *taskpb.GetTasksRequest) (*taskpb.GetTasksResponse, error) {
	logrus.Infof("user: %s, getting tasks", req.UserName)
	res := &taskpb.GetTasksResponse{}

	errUserMessage, statusGRPC, tasks, err := t.repo.GetTasks(req)
	if err != nil {
		logrus.Errorf("Can`t get tasks, user: %s", req.UserName)
		statusGRPC.Message = err.Error()
		res.Status = statusGRPC
		res.UserErrorMessage = errUserMessage
		return res, nil
	}
	respTasks := make([]*taskpb.Task, 0, len(tasks))
	for i := range tasks {
		respTasks = append(respTasks, &tasks[i])
	}

	res.Status = statusGRPC
	res.UserErrorMessage = errUserMessage
	res.Task = respTasks
	return res, nil
}

func (t *TaskServer) DeleteTask(ctx context.Context, req *taskpb.DeleteTaskRequest) (*taskpb.DeleteTaskResponse, error) {
	logrus.Info("user %s, start delete task%s", req.UserName, req.TaskNumber)
	res := &taskpb.DeleteTaskResponse{}

	errUserMessage, statusGRPC, err := t.repo.DeleteTask(req.UserName, int(req.TaskNumber))
	if err != nil {
		logrus.Errorf("Can`t delete task, user: %s", req.UserName)
		statusGRPC.Message = err.Error()
		res.Status = statusGRPC
		res.UserErrorMessage = errUserMessage
		return res, nil
	}

	res.Status = statusGRPC
	res.UserErrorMessage = errUserMessage
	return res, nil
}

func (t *TaskServer) ChangeTask(ctx context.Context, req *taskpb.ChangeTaskRequest) (*taskpb.ChangeTaskResponse, error) {
	logrus.Info("user %s, start change task%s", req.UserName, req.TaskNum)
	res := &taskpb.ChangeTaskResponse{}
	errUserMessage, statusGRPC, err := t.repo.ChangeTask(req)
	if err != nil {
		logrus.Errorf("Can`t change task, user: %s", req.UserName)
		statusGRPC.Message = err.Error()
		res.Status = statusGRPC
		res.UserErrorMessage = errUserMessage
		return res, nil
	}

	res.Status = statusGRPC
	res.UserErrorMessage = errUserMessage
	return res, nil
}
