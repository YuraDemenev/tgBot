package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"tgbot/bot-service/protoGenFiles/tgBot/bot-service/protoGenFiles/health"
	"tgbot/bot-service/protoGenFiles/tgBot/bot-service/protoGenFiles/taskpb"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func AddTaskService(userText string) error {
	stringsArr := strings.Split(userText, ",")
	if len(stringsArr) != 4 {
		err := fmt.Errorf("AddTaskService, stringsArr has length:%b", len(stringsArr))
		return err
	}

	// Clear strings
	for i, v := range stringsArr {
		stringsArr[i] = strings.TrimSpace(v)
	}

	task := &taskpb.Task{}
	task.Name = stringsArr[0]
	task.Description = stringsArr[1]

	// Work with date
	dateArrString := strings.Split(stringsArr[2], ".")
	if len(dateArrString) != 3 {
		err := fmt.Errorf("AddTaskService, dateArrString has length:%b", len(dateArrString))
		return err
	}
	dateArr := make([]int32, 3)
	for i, v := range dateArrString {
		integer, err := strconv.Atoi(v)
		if err != nil {
			err := fmt.Errorf("AddTaskService, can`t convert str to int:%s", v)
			return err
		}
		dateArr[i] = int32(integer)
	}

	task.Date = &taskpb.MyDate{
		Day:   dateArr[0],
		Month: dateArr[1],
		Year:  dateArr[2],
	}

	// Work with time
	parsedTime, err := time.Parse(time.RFC3339, stringsArr[3])
	if err != nil {
		logrus.Errorf("AddTaskService, can`t parse time err: %w", err)
		return err
	}
	task.Time = timestamppb.New(parsedTime)
	return nil
}

func sendTaskGRPC(task *taskpb.Task) (bool, error) {
	err := healthCheck()
	if err != nil {
		return false, err
	}

	// Create grpc connection to task-service
	conn, err := grpc.NewClient("localhost:50002", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logrus.Errorf("sendTaskGRPC, failed to connect to task-service: %v", err)
		return false, err
	}
	defer conn.Close()

	// create grpc client
	client := taskpb.NewTaskServiceClient(conn)

	ctx := context.Background()
	resp, err := client.SendTask(ctx, task)
	if err != nil {
		logrus.Errorf("sendTaskGRPC, failed to send task via gRPC: %v", err)
		return false, err
	}

	logrus.Infof("Task sent successfully to task-service, response: %v", resp.Ok)
	return resp.Ok, nil
}

func healthCheck() error {
	//Подготавливаем Health check
	conn, err := grpc.NewClient("localhost:50001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logrus.Errorf("healthCheck can`t check health, err:%v", err)
		return err
	}
	defer conn.Close()
	healthClient := health.NewHealthClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	//Делаем health check
	resp, err := healthClient.Check(ctx, &health.HealthCheckRequest{Service: "task.TaskService"})
	if err != nil {
		logrus.Errorf("healthCheck, Health check failed : %v", err)
		return err
	}

	// Сервер готов, можно отправлять задачу
	if resp.Status == health.HealthCheckResponse_SERVING {
		logrus.Errorf("healthCheck, health resp got status: %s", resp.Status.String())
		return nil
	}

	return fmt.Errorf("healthCheck, task-service not ready")
}
