package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"tgbot/bot-service/protoGenFiles/tgBot/bot-service/protoGenFiles/taskpb"
	"tgbot/task-service/internal/cache"
	"tgbot/task-service/internal/handlers"
	"tgbot/task-service/internal/rabbitmq"
	"tgbot/task-service/internal/repositories"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func startGRPCServer(ctx context.Context, wg *sync.WaitGroup, repo *repositories.Repository) {
	defer wg.Done()
	// add a listener address
	lis, err := net.Listen("tcp", ":50002")
	if err != nil {
		logrus.Fatalf("ERROR STARTING THE SERVER : %v", err)
	}

	// create the grpc server
	grpcServer := grpc.NewServer()
	taskpb.RegisterTaskServiceServer(grpcServer, handlers.NewTaskServer(repo))

	// Register health service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// start serving to the address
	go func() {
		if err := grpcServer.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			logrus.Fatalf("grpcServer can't serve: %v", err)
		}
	}()

	select {
	case <-ctx.Done():
		grpcServer.GracefulStop()
		return
	}
}

func main() {
	//Prepare for starting
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var mainWG sync.WaitGroup

	logrus.SetFormatter(new(logrus.JSONFormatter))
	err := initConfig()
	if err != nil {
		logrus.Fatalf("initializing config error: %s", err.Error())
	}

	//connect to DB
	db, err := repositories.NewPostgresDB(repositories.Config{
		Host:     viper.GetString("db.host"),
		Port:     viper.GetString("db.port"),
		UserName: viper.GetString("db.username"),
		DBName:   viper.GetString("db.dbname"),
		SSLMode:  viper.GetString("db.sslmode"),
		Password: viper.GetString("db.password"),
	})
	if err != nil {
		logrus.Fatalf("failed to initialize db: %s", err.Error())
	}

	//connect to redis
	redisCache := cache.NewRedisCache(cache.RedisConfig{
		Host: viper.GetString("redis.host"),
		DB:   viper.GetInt("redis.db"),
	})

	//Init rabbitMQ
	r := rabbitmq.NewRabbitMQ("amqp://guest:guest@localhost:5672/")
	defer r.Close()

	r.DeclareQueue(rabbitmq.DelayedExchange, rabbitmq.NotifyTaskQueue, rabbitmq.NotifyKey)

	//init repository
	repo := repositories.NewRepository(db, redisCache, r)

	//Start GRPC server
	mainWG.Add(1)
	go startGRPCServer(ctx, &mainWG, repo)
	logrus.Info("Service task-service started grpc server")

	//handle graceful shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		logrus.Infof("Received shutdown signal: %v", sig)
		cancel()
	}()

	logrus.Info("Service task-service started")
	mainWG.Wait()
}

func initConfig() error {
	viper.AddConfigPath("../config")
	viper.SetConfigName("config")
	return viper.ReadInConfig()
}
