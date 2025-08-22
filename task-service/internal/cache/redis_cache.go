package cache

import (
	"context"
	"strconv"
	"tgbot/bot-service/protoGenFiles/tgBot/bot-service/protoGenFiles/taskpb"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

// Интерфейс для кэша
type Cache interface {
	SetTask(task *taskpb.Task, taskID int)
	GetTask(taskID int) *taskpb.Task
	DeleteTask(taskID int)
	GetTasks(tasksID []int) ([]*taskpb.Task, []int, error)
}

// Реализация через Redis
type RedisCache struct {
	client *redis.Client
}

type RedisConfig struct {
	Host string
	DB   int
}

func NewRedisCache(redisConfig RedisConfig) *RedisCache {
	client := redis.NewClient(&redis.Options{
		Addr:     redisConfig.Host,
		Password: "",
		DB:       redisConfig.DB,
	})
	return &RedisCache{client: client}
}

func (r *RedisCache) SetTask(task *taskpb.Task, taskID int) {
	ctx := context.Background()
	//Prepare data
	data, err := proto.Marshal(task)
	if err != nil {
		logrus.Errorf("SetTask, can`t marshal task, err:%v", err)
		return
	}

	_, err = r.client.Set(ctx, strconv.Itoa(taskID), data, time.Hour).Result()
	if err != nil {
		logrus.Errorf("SetTask, redis can`t save task, taskID:%d, task:%v, err:%v", taskID, task, err)
		return
	}
}

func (r *RedisCache) GetTask(taskID int) *taskpb.Task {
	ctx := context.Background()

	res, err := r.client.Get(ctx, strconv.Itoa(taskID)).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		logrus.Errorf("error in redis when get task: %v", err)
		return nil
	}

	var task taskpb.Task
	err = proto.Unmarshal([]byte(res), &task)
	if err != nil {
		logrus.Errorf("error in redis when unmarshal task: %v", err)
		return nil
	}
	return &task
}

func (r *RedisCache) DeleteTask(taskID int) {
	ctx := context.Background()

	_, err := r.client.Del(ctx, strconv.Itoa(taskID)).Result()
	if err == redis.Nil {
		return
	}
	if err != nil {
		logrus.Errorf("error in redis when delete task: %v", err)
		return
	}

	return
}

func (r *RedisCache) GetTasks(tasksID []int) ([]*taskpb.Task, []int, error) {
	ctx := context.Background()
	tasksIDStrings := make([]string, len(tasksID))
	missingTasks := make([]int, 0, len(tasksID))
	resultTask := make([]*taskpb.Task, 0)

	for i, val := range tasksID {
		tasksIDStrings[i] = strconv.Itoa(val)
	}

	vals, err := r.client.MGet(ctx, tasksIDStrings...).Result()
	if err != nil {
		logrus.Errorf("redis GetTasks, can`t do mget, err%v", err)
		return nil, nil, err
	}

	for i, v := range vals {
		if v == nil {
			missingTasks = append(missingTasks, tasksID[i])
		} else {
			val, ok := v.(string)
			if !ok {
				logrus.Errorf("redis GetTasks, unexpected type for value: %T", v)
				continue
			}
			bytes := []byte(val)
			var task taskpb.Task

			if err := proto.Unmarshal(bytes, &task); err != nil {
				logrus.Errorf("redis GetTasks, can`t convert value to task, err:%v", err)
				missingTasks = append(missingTasks, tasksID[i])
				continue
			}
			resultTask = append(resultTask, &task)
		}
	}

	return resultTask, missingTasks, nil
}
