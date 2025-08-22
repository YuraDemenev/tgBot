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
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TasksPostgres struct {
	db          *sqlx.DB
	cacheClient cache.Cache
}

func NewTasksPostgres(db *sqlx.DB, cache cache.Cache) Tasks {
	return &TasksPostgres{db: db, cacheClient: cache}
}

func (t *TasksPostgres) ChangeTask(req *taskpb.ChangeTaskRequest) error {
	//Check correct values
	newValueStr := req.NewValue
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
			return err
		}

		dateArrInt := make([]int, 3)
		for i, v := range dateArrStr {
			integer, err := strconv.Atoi(v)
			if err != nil {
				logrus.Errorf("changeTask, can`t conver date string:%s to int, err:%v", v, err)
				return err
			}
			dateArrInt[i] = integer
		}

		date := time.Date(dateArrInt[2], time.Month(dateArrInt[1]), dateArrInt[0], 0, 0, 0, 0, time.UTC)
		args = append(args, date)
		query = `UPDATE tasks SET date = $1 WHERE id=$2`

	case "Time":
		parsedTime, err := time.Parse("15:04", newValueStr)
		if err != nil {
			logrus.Errorf("changeTask, can`t parse time:%s, err:%v", newValueStr, err)
			return err
		}

		if parsedTime.Before(time.Now()) {
			err = fmt.Errorf("changeTask, parsedTime %v is before current time %v", parsedTime, time.Now())
			logrus.Errorf(err.Error())
			return err
		}
		query = `UPDATE tasks SET time = $1 WHERE id = $2`
		args = append(args, parsedTime)
	default:
		err := fmt.Errorf("changeTask, uknown change value %s", req.ChangeValue)
		return err
	}

	tx, err := t.db.Begin()
	if err != nil {
		logrus.Errorf("changeTask, can`t prepare for transaction err:%v", err)
		return err
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
		return err
	}

	args = append(args, taskID)
	_, err = tx.Exec(query, args...)
	if err != nil {
		logrus.Errorf("changeTask, Can`t make UPDATE query, err:%v", err)
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

func (t *TasksPostgres) DeleteTask(userName string, taskNum int) error {
	userHash := getUserHash(userName)
	tx, err := t.db.Begin()
	if err != nil {
		logrus.Errorf("deleteTask, can`t prepare for transaction err:%v", err)
		return err
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
		return err
	}

	_, err = tx.Exec(`
		DELETE FROM tasks
		WHERE id = $1;`, taskID)
	if err != nil {
		logrus.Errorf("deleteTask, can`t delete task:%s, err:%v", taskNum, err)
		tx.Rollback()
		return err
	}

	tx.Commit()
	t.cacheClient.DeleteTask(taskID)
	return nil
}

func (t *TasksPostgres) GetTasks(req *taskpb.GetTasksRequest) ([]taskpb.Task, error) {
	logrus.Infof("Start get tasks for user: %s", req.UserName)
	userHash := getUserHash(req.UserName)
	userTasks := make([]taskpb.Task, 0)

	//Get ids
	rows, err := t.db.QueryRow(`SELECT t.id
	FROM tasks as t
	JOIN users u on u.id = t.user_id
	WHERE u.user_name_hash = $1;
	`, userHash)

	if err != nil {
		logrus.Errorf("getTasks, can`t query err:%v", err)
		return nil, err
	}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			logrus.Errorf("getTasks, can`t scan id, err:%v", err)
			return nil, err
		}

		//Check Redis for task
		redisTask := t.cacheClient.GetTask(id)
		if redisTask != nil {
			userTasks = append(userTasks, *redisTask)
			continue
		}

		//if no taks in redis get task by id
		var task taskpb.Task
		var date time.Time
		var myTime time.Time

		taskRow := t.db.QueryRow("SELECT t.task_name, t.description, t.date, t.time FROM tasks as t WHERE t.id = $1", id)
		if err := taskRow.Scan(&task.Name, &task.Description, &date, &myTime); err != nil {
			logrus.Errorf("getTasks, can`t scan task:%v", err)
			return nil, err
		}
		task.Date = &taskpb.MyDate{
			Day:   int32(date.Day()),
			Month: int32(date.Month()),
			Year:  int32(date.Year()),
		}
		task.Time = timestamppb.New(myTime)
		userTasks = append(userTasks, task)
	}

	return userTasks, nil
}

func (t *TasksPostgres) SaveTask(req *taskpb.SendTaskRequest) error {
	logrus.Infof("Start save task for user: %s", req.UserName)
	task := req.Task
	userHash := getUserHash(req.UserName)

	tx, err := t.db.Begin()
	if err != nil {
		logrus.Errorf("can`t prepare for transaction err:%v", err)
		return err
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
				logrus.Errorf("Can`t scan userID, after add user:%s err:%v", req.UserName, err)
				return err
			}

			// If user exist
		} else {
			tx.Rollback()
			logrus.Errorf("Can`t scan userID, err:%v", err)
			return err
		}
	}

	// Add task
	var taskID int
	date := time.Date(int(task.Date.Year), time.Month(task.Date.Month), int(task.Date.Day), 0, 0, 0, 0, time.UTC)
	myTime := task.Time.AsTime()
	row = tx.QueryRow(`
		INSERT INTO tasks (user_id, task_name, description, date, time)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`,
		userID, task.Name, task.Description, date, myTime)

	err = row.Scan(&taskID)
	if err != nil {
		tx.Rollback()
		logrus.Errorf("Can`t insert task userID:%d, err:%v", userID, err)
		return err
	}
	tx.Commit()

	//Set task to redis
	t.cacheClient.SetTask(task, taskID)
	return nil
}

func getUserHash(userName string) string {
	salt := "akhljmb=sd23"

	h := sha256.New()
	h.Write([]byte(userName))

	return hex.EncodeToString(h.Sum([]byte(salt)))
}
