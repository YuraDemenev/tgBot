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
)

type TasksPostgres struct {
	db          *sqlx.DB
	cacheClient cache.Cache
}

func NewTasksPostgres(db *sqlx.DB, cache cache.Cache) Tasks {
	return &TasksPostgres{db: db, cacheClient: cache}
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
