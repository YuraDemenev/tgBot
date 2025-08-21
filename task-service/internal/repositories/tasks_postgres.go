package repositories

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
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

func (t *TasksPostgres) DeleteTask(userName string, taskNum int) error {
	userHash := getUserHash(userName)
	tx, err := t.db.Begin()
	if err != nil {
		logrus.Errorf("DeleteTask, can`t prepare for transaction err:%v", err)
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
		logrus.Errorf("DeleteTask, Can`t scan id, err:%v", err)
		return err
	}

	_, err = tx.Exec(`
		DELETE FROM tasks
		WHERE id = $1;`, taskID)
	if err != nil {
		logrus.Errorf("DeleteTask, can`t delete task:%s, err:%v", taskNum, err)
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
	rows, err := t.db.Query(`SELECT t.id
	FROM tasks as t
	JOIN users u on u.id = t.user_id
	WHERE u.user_name_hash = $1;
	`, userHash)

	if err != nil {
		logrus.Errorf("GetTasks, can`t query err:%v", err)
		return nil, err
	}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			logrus.Errorf("GetTasks, can`t scan id, err:%v", err)
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
			logrus.Errorf("GetTasks, can`t scan task:%v", err)
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
