package repositories

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"tgbot/bot-service/protoGenFiles/tgBot/bot-service/protoGenFiles/taskpb"
	"tgbot/task-service/internal/cache"
	"tgbot/task-service/internal/rabbitmq"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TasksPostgres struct {
	db          *sqlx.DB
	cacheClient cache.Cache
	r           *rabbitmq.RabbitMQ
}

func NewTasksPostgres(db *sqlx.DB, cache cache.Cache, r *rabbitmq.RabbitMQ) Tasks {
	return &TasksPostgres{db: db, cacheClient: cache, r: r}
}

func (t *TasksPostgres) ChangeTask(req *taskpb.ChangeTaskRequest) (string, *status.Status, error) {
	//Check correct values
	newValueStr := req.NewValue
	errUserMessage := ""
	status := &status.Status{}
	var query string
	args := make([]interface{}, 0, 2)

	switch req.ChangeValue {
	case "Task Name":
		query = `UPDATE tasks SET task_name = $1 WHERE id=$2`
		args = append(args, newValueStr)

	case "Description":
		query = `UPDATE tasks SET description = $1 WHERE id=$2`
		args = append(args, newValueStr)

	case "Date":
		dateArrStr := strings.Split(newValueStr, ".")
		if len(dateArrStr) != 3 {
			err := fmt.Errorf("changeTask, date does`t have 3 elements")
			logrus.Errorf(err.Error())
			errUserMessage := "Вы ввели неверную дату, пожалуйста введите верное значение"
			status.Code = int32(codes.InvalidArgument)
			return errUserMessage, status, err
		}

		dateArrInt := make([]int, 3)
		for i, v := range dateArrStr {
			integer, err := strconv.Atoi(v)
			if err != nil {
				logrus.Errorf("changeTask, can`t conver date string:%s to int, err:%v", v, err)
				errUserMessage := "Вы ввели неверную дату, одно из значений не является чилсом, пожалуйста введите верное значение"
				status.Code = int32(codes.InvalidArgument)
				return errUserMessage, status, err
			}
			dateArrInt[i] = integer
		}

		//TODO добавить проверку даты
		date, err := validTime(dateArrInt[0], dateArrInt[1], dateArrInt[2])
		if err != nil {
			logrus.Errorf("changeTask, can`t create date, err:%v ", err)
			errUserMessage = "Вы ввели неккоректную дату, пожалуйста напишите существующую дату"
			status.Code = int32(codes.InvalidArgument)
			return errUserMessage, status, err
		}

		// Check time before
		if date.Before(time.Now()) {
			err := fmt.Errorf("changeTask date before time now")
			logrus.Errorf("SaveTask, err:%v", err)
			errUserMessage = "Вы ввели дату, которая уже прошла, пожалуйста ввидете дату с будующим временем"
			status.Code = int32(codes.InvalidArgument)
			return errUserMessage, status, err
		}

		// Check time after
		if date.After(time.Now().AddDate(0, 2, 0)) {
			err := fmt.Errorf("user date after two months")
			logrus.Errorf("changeTask, err:%v ", err)
			errUserMessage = "Вы ввели дату, которая превышает допустимый предел, пожалуйста ввидете дату максимум через 2 месяца"
			status.Code = int32(codes.InvalidArgument)
			return errUserMessage, status, err
		}

		args = append(args, date)
		query = `UPDATE tasks SET date = $1 WHERE id=$2`

	case "Time":
		//TODO убрать проверку времени
		parsedTime, err := time.Parse("15:04", newValueStr)
		if err != nil {
			logrus.Errorf("changeTask, can`t parse time:%s, err:%v", newValueStr, err)
			errUserMessage = "Вы ввели неккоректное время, пожалуйста напишите существующие время"
			status.Code = int32(codes.InvalidArgument)
			return errUserMessage, status, err
		}

		query = `UPDATE tasks SET time = $1 WHERE id = $2`
		args = append(args, parsedTime)
	default:
		err := fmt.Errorf("changeTask, uknown change value %s", req.ChangeValue)
		errUserMessage = "Произошла ошибка на стороне сервера, пожалуйста попробуйте ещё раз через некоторое время"
		status.Code = int32(codes.Internal)
		return errUserMessage, status, err
	}

	tx, err := t.db.Begin()
	if err != nil {
		logrus.Errorf("changeTask, can`t prepare for transaction err:%v", err)
		errUserMessage = "Произошла ошибка на стороне сервера, пожалуйста попробуйте ещё раз через некоторое время"
		status.Code = int32(codes.Internal)
		return errUserMessage, status, err
	}

	var taskID int
	row := t.db.QueryRow(`
		SELECT id FROM tasks
		WHERE user_id = (SELECT id FROM users WHERE user_name_hash = $1)
		ORDER BY id
		OFFSET $2 LIMIT 1`, getUserHash(req.UserName), req.TaskNum-1)

	err = row.Scan(&taskID)
	if err != nil {
		logrus.Errorf("changeTask, Can`t scan id, err:%v", err)
		errUserMessage = "Вы ввели неверный номер задачи, такой задачи у вас нет, введите пожалуйста корректный номер"
		status.Code = int32(codes.NotFound)
		return errUserMessage, status, err
	}

	args = append(args, taskID)
	_, err = tx.Exec(query, args...)
	if err != nil {
		logrus.Errorf("changeTask, Can`t make UPDATE query, err:%v", err)
		tx.Rollback()
		errUserMessage = "Произошла ошибка на стороне сервера, пожалуйста попробуйте ещё раз через некоторое время"
		status.Code = int32(codes.Internal)
		return errUserMessage, status, err
	}

	tx.Commit()
	return errUserMessage, status, nil
}

func (t *TasksPostgres) DeleteTask(userName string, taskNum int) (string, *status.Status, error) {
	userHash := getUserHash(userName)
	errUserMessage := ""
	status := &status.Status{}

	tx, err := t.db.Begin()
	if err != nil {
		logrus.Errorf("deleteTask, can`t prepare for transaction err:%v", err)
		errUserMessage = "Произошла ошибка на стороне сервера, пожалуйста попробуйте ещё раз через некоторое время"
		status.Code = int32(codes.Internal)
		return errUserMessage, status, err
	}
	var taskID int

	row := t.db.QueryRow(`
		SELECT id FROM tasks
		WHERE user_id = (SELECT id FROM users WHERE user_name_hash = $1)
		ORDER BY id
		OFFSET $2 LIMIT 1`, userHash, taskNum-1)

	err = row.Scan(&taskID)
	if err != nil {
		logrus.Errorf("deleteTask, Can`t scan id, err:%v", err)
		errUserMessage = "Вы ввели неверный номер задачи, такой задачи у вас нет, введите пожалуйста корректный номер"
		status.Code = int32(codes.NotFound)
		return errUserMessage, status, err
	}

	_, err = tx.Exec(`
		DELETE FROM tasks
		WHERE id = $1;`, taskID)
	if err != nil {
		logrus.Errorf("deleteTask, can`t delete task:%s, err:%v", taskNum, err)
		errUserMessage = "Произошла ошибка на стороне сервера, пожалуйста попробуйте ещё раз через некоторое время"
		status.Code = int32(codes.Internal)
		tx.Rollback()
		return errUserMessage, status, err
	}

	tx.Commit()
	t.cacheClient.DeleteTask(taskID)
	status.Code = int32(codes.OK)
	return errUserMessage, status, nil
}

func (t *TasksPostgres) GetTasks(req *taskpb.GetTasksRequest) (string, *status.Status, []taskpb.Task, error) {
	logrus.Infof("Start get tasks for user: %s", req.UserName)
	userHash := getUserHash(req.UserName)
	userTasks := make([]taskpb.Task, 0)
	errUserMessage := ""
	status := &status.Status{}

	//Get ids
	rows, err := t.db.Query(`SELECT t.id
	FROM tasks as t
	JOIN users u on u.id = t.user_id
	WHERE u.user_name_hash = $1;
	`, userHash)
	if err != nil {
		logrus.Errorf("getTasks, can`t query err:%v", err)
		errUserMessage = "Произошла ошибка на стороне сервера, пожалуйста попробуйте ещё раз через некоторое время"
		status.Code = int32(codes.Internal)
		return errUserMessage, status, nil, err
	}
	defer rows.Close()
	ids := make([]int, 0)

	for rows.Next() {
		var id int
		err := rows.Scan(&id)
		if err != nil {
			logrus.Errorf("getTasks, can`t scan err:%v", err)
			errUserMessage = "Произошла ошибка на стороне сервера, пожалуйста попробуйте ещё раз через некоторое время"
			status.Code = int32(codes.Internal)
			return errUserMessage, status, nil, err
		}
		ids = append(ids, id)
	}

	// If user has 0 tasks
	if len(ids) == 0 {
		err := fmt.Errorf("getTasks, user has 0 tasks")
		logrus.Error(err)
		errUserMessage = "У тебя нет ни 1 созданного уведомления, пожалуйста сначала создай уведомление"
		status.Code = int32(codes.NotFound)
		return errUserMessage, status, nil, err
	}

	//Check Redis for task
	redisTasks, missingTasks, err := t.cacheClient.GetTasks(ids)
	if err != nil {
		errUserMessage = "Произошла ошибка на стороне сервера, пожалуйста попробуйте ещё раз через некоторое время"
		status.Code = int32(codes.Internal)
		return errUserMessage, status, nil, err
	}

	for _, v := range redisTasks {
		userTasks = append(userTasks, *v)
	}

	//Create query with in
	query, args, err := sqlx.In("SELECT t.task_name, t.description, t.date, t.time FROM tasks as t WHERE t.id in (?)", missingTasks)
	if err != nil {
		logrus.Errorf("getTasks, can`t create query with in parametr, err%v", err)
		errUserMessage = "Произошла ошибка на стороне сервера, пожалуйста попробуйте ещё раз через некоторое время"
		status.Code = int32(codes.Internal)
		return errUserMessage, status, nil, err
	}

	query = t.db.Rebind(query)

	rows, err = t.db.Query(query, args...)
	if err != nil {
		logrus.Errorf("getTasks, can`t create query err:%v", err)
		errUserMessage = "Произошла ошибка на стороне сервера, пожалуйста попробуйте ещё раз через некоторое время"
		status.Code = int32(codes.Internal)
		return errUserMessage, status, nil, err
	}

	//TODO нужно ли здесь defer close?
	//Get other tasks from db
	defer rows.Close()
	for rows.Next() {
		var task taskpb.Task
		var date time.Time
		var myTime time.Time

		if err := rows.Scan(&task.Name, &task.Description, &date, &myTime); err != nil {
			logrus.Errorf("getTasks, can`t scan task:%v", err)
			errUserMessage = "Произошла ошибка на стороне сервера, пожалуйста попробуйте ещё раз через некоторое время"
			status.Code = int32(codes.Internal)
			return errUserMessage, status, nil, err
		}

		task.Date = &taskpb.MyDate{
			Day:   int32(date.Day()),
			Month: int32(date.Month()),
			Year:  int32(date.Year()),
		}
		task.Time = timestamppb.New(myTime)
		userTasks = append(userTasks, task)
	}

	status.Code = int32(codes.OK)
	return errUserMessage, status, userTasks, nil
}

func (t *TasksPostgres) SaveTask(req *taskpb.SendTaskRequest) (string, *status.Status, error) {
	logrus.Infof("Start save task for user: %s", req.UserName)
	errUserMessage := ""
	status := &status.Status{}
	// &status.Status{Code: int32(codes.OK)}
	task := req.Task
	userHash := getUserHash(req.UserName)

	tx, err := t.db.Begin()
	if err != nil {
		logrus.Errorf("SaveTask, can`t prepare for transaction err:%v", err)
		errUserMessage = "Произошла ошибка на стороне сервера, пожалуйста попробуйте ещё раз через некоторое время"
		status.Code = int32(codes.Internal)
		return errUserMessage, status, err
	}

	var userID int
	// First check user
	row := tx.QueryRow("SELECT id FROM users WHERE user_name_hash=$1", userHash)
	err = row.Scan(&userID)
	if err != nil {
		// If no such user add user
		if errors.Is(err, sql.ErrNoRows) {
			row = tx.QueryRow("INSERT INTO users (user_name_hash) VALUES ($1) RETURNING id", userHash)
			err = row.Scan(&userID)
			if err != nil {
				tx.Rollback()
				errUserMessage = "Произошла ошибка на стороне сервера, пожалуйста попробуйте ещё раз через некоторое время"
				status.Code = int32(codes.Internal)
				logrus.Errorf("SaveTask, Can`t scan userID, after add user:%s err:%v", req.UserName, err)
				return errUserMessage, status, err
			}

			// If user exist
		} else {
			tx.Rollback()
			logrus.Errorf("SaveTask, Can`t scan userID, err:%v", err)
			errUserMessage = "Произошла ошибка на стороне сервера, пожалуйста попробуйте ещё раз через некоторое время"
			status.Code = int32(codes.Internal)
			return errUserMessage, status, err
		}
	}

	// Add task
	var taskID int
	// Check valid time
	date, err := validTime(int(task.Date.Day), int(task.Date.Month), int(task.Date.Year))
	if err != nil {
		logrus.Errorf("SaveTask, can`t create date, err:%v ", err)
		errUserMessage = "Вы ввели неккоректную дату, пожалуйста напишите существующую дату"
		status.Code = int32(codes.InvalidArgument)
		return errUserMessage, status, err
	}

	// Check time before
	if date.Before(time.Now()) {
		err := fmt.Errorf("user date before time now")
		logrus.Errorf("SaveTask, err:%v", err)
		errUserMessage = "Вы ввели дату, которая уже прошла, пожалуйста ввидете дату с будующим временем"
		status.Code = int32(codes.InvalidArgument)
		return errUserMessage, status, err
	}

	// Check time after
	if date.After(time.Now().AddDate(0, 2, 0)) {
		err := fmt.Errorf("user date after two months")
		logrus.Errorf("SaveTask, err:%v ", err)
		errUserMessage = "Вы ввели дату, которая превышает допустимый предел, пожалуйста ввидете дату максимум через 2 месяца"
		status.Code = int32(codes.InvalidArgument)
		return errUserMessage, status, err
	}

	myTime := task.Time.AsTime()
	row = tx.QueryRow(`
		INSERT INTO tasks (user_id, task_name, description, date, time)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`,
		userID, task.Name, task.Description, date, myTime)

	err = row.Scan(&taskID)
	if err != nil {
		tx.Rollback()
		logrus.Errorf("SaveTask, Can`t insert task userID:%d, err:%v", userID, err)
		errUserMessage = "Произошла ошибка на стороне сервера, пожалуйста попробуйте ещё раз через некоторое время"
		status.Code = int32(codes.Internal)
		return errUserMessage, status, err
	}
	tx.Commit()

	//Set task to redis
	t.cacheClient.SetTask(task, taskID)
	status.Code = int32(codes.OK)
	return errUserMessage, status, nil
}

func getUserHash(userName string) string {
	salt := "akhljmb=sd23"

	h := sha256.New()
	h.Write([]byte(userName))

	return hex.EncodeToString(h.Sum([]byte(salt)))
}

func validTime(day, month, year int) (time.Time, error) {
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	//validate time because go auto change wrong time

	if t.Day() != day || int(t.Month()) != month || t.Year() != year {
		return time.Time{}, fmt.Errorf("user send wrong date: %02d.%02d.%d", day, month, year)
	}
	return t, nil
}
