package repositories

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"tgbot/bot-service/protoGenFiles/tgBot/bot-service/protoGenFiles/taskpb"
	"tgbot/task-service/internal/cache"

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
			row = tx.QueryRow("INSERT INT users (user_name_hash, count_tasks) VALUES $1,$2 RETURNING id", userHash, 0)
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
	_, err = tx.Exec(`
		INSERT INTO tasks (user_id, task_name, description, date, time)
		VALUES ($1, $2, $3, $4, $5)`,
		userID, task.Name, task.Description, task.Date, task.Time)

	if err != nil {
		tx.Rollback()
		logrus.Errorf("Can`t insert task userID:%d, err:%v", userID, err)
		return err
	}
	return nil
}

func getUserHash(userName string) string {
	salt := "akhljmb=sd23"

	h := sha256.New()
	h.Write([]byte(userName))

	return hex.EncodeToString(h.Sum([]byte(salt)))
}
