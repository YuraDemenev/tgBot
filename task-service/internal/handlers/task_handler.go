package handlers

import (
	"context"
	"fmt"
	"tgbot/bot-service/protoGenFiles/tgBot/bot-service/protoGenFiles/taskpb"

	"github.com/sirupsen/logrus"
	"google.golang.org/genproto/googleapis/rpc/status"
)

type TaskServer struct {
	taskpb.UnimplementedTaskServiceServer
}

func (t *TaskServer) SendTask(ctx context.Context, req *taskpb.SendTaskRequest) (*status.Status, error) {
	logrus.Infof("user: %s, send task", req.UserName)
	fmt.Printf("Got task: %+v\n", req)
	//TODO change status
	return &status.Status{}, nil
}
