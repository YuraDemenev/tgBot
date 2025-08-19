package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"tgbot/bot-service/protoGenFiles/tgBot/bot-service/protoGenFiles/taskpb"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func SendTaskGRPC(userText string, userName string) error {
	logrus.Infof("started SendTaskGRPC for user:%s", userName)

	task, err := createTask(userText)
	if err != nil {
		return err
	}

	// Do health check
	err = healthCheck()
	if err != nil {
		return err
	}
	// Create grpc connection to task-service
	conn, err := grpc.NewClient("localhost:50002", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logrus.Errorf("sendTaskGRPC, failed to connect to task-service: %v", err)
		return err
	}
	defer conn.Close()

	client := taskpb.NewTaskServiceClient(conn)

	resp, err := client.SendTask(context.Background(), &taskpb.SendTaskRequest{Task: task, UserName: userName})
	if err != nil {
		logrus.Errorf("SendTaskGRPC, can`t send task err: %v", err)
	}
	fmt.Println(resp)
	return nil
}

func createTask(userText string) (*taskpb.Task, error) {
	stringsArr := strings.Split(userText, ",")
	if len(stringsArr) != 4 {
		err := fmt.Errorf("AddTaskService, stringsArr has length:%b", len(stringsArr))
		return nil, err
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
		return nil, err
	}
	dateArr := make([]int32, 3)
	for i, v := range dateArrString {
		integer, err := strconv.Atoi(v)
		if err != nil {
			err := fmt.Errorf("AddTaskService, can`t convert str to int:%s", v)
			return nil, err
		}
		dateArr[i] = int32(integer)
	}

	task.Date = &taskpb.MyDate{
		Day:   dateArr[0],
		Month: dateArr[1],
		Year:  dateArr[2],
	}

	// Work with time
	parsedTime, err := time.Parse("15:05", stringsArr[3])
	if err != nil {
		logrus.Errorf("AddTaskService, can`t parse time err: %w", err)
		return nil, err
	}
	task.Time = timestamppb.New(parsedTime)

	return task, nil
}

func healthCheck() error {
	//Prepare Health check
	conn, err := grpc.NewClient("localhost:50002", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logrus.Errorf("healthCheck can`t create new client, err:%v", err)
		return err
	}
	defer conn.Close()
	healthClient := grpc_health_v1.NewHealthClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	//Do health check
	resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		logrus.Errorf("healthCheck, Health check failed : %v", err)
		return err
	}

	if resp.Status == grpc_health_v1.HealthCheckResponse_SERVING {
		logrus.Infof("healthCheck, health resp got status: %s", resp.Status.String())
		return nil
	}

	return fmt.Errorf("healthCheck, task-service not ready")
}
