package handlers

import (
	"context"
	"fmt"
	"tgbot/bot-service/protoGenFiles/tgBot/bot-service/protoGenFiles/taskpb"

	"github.com/sirupsen/logrus"
)

type TaskServer struct {
	taskpb.UnimplementedTaskServiceServer
}

func (s *TaskServer) ReceivedTask(ctx context.Context, task *taskpb.Task) (*taskpb.SendTaskResponse, error) {
	// Формируем строку с содержимым задачи
	logMessage := fmt.Sprintf(
		"Received task: Name=%s, Description=%s, Date=%d.%d.%d, Time=%s",
		task.Name,
		task.Description,
		task.Date.Day, task.Date.Month, task.Date.Year,
		task.Time.AsTime().Format("15:04"),
	)

	// Выводим содержимое задачи в лог
	logrus.Info(logMessage)

	// Возвращаем успешный ответ
	return &taskpb.SendTaskResponse{Ok: true}, nil
}
