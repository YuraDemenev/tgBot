package services

import (
	"fmt"
	"strconv"
	"strings"
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

}

func sendTaskGRPC(task *taskpb.Task) (bool, error) {
	conn, err := grpc.NewClient("localhost:5001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {

	}
	defer conn.Close()
}
