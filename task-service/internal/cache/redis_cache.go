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
