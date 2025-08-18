package handlers

import (
	"context"
	"fmt"
	"tgbot/bot-service/protoGenFiles/tgBot/bot-service/protoGenFiles/taskpb"

	"google.golang.org/genproto/googleapis/rpc/status"
)

type TaskServer struct {
	taskpb.UnimplementedTaskServiceServer
}

func (t *TaskServer) sendTask(ctx context.Context, req *taskpb.SendTaskRequest) (*status.Status, error) {
	fmt.Printf("Got task: %+v\n", req)
	//TODO change status
	return &status.Status{}, nil
}
