package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"tgbot/bot-service/internal/handlers"
	"tgbot/bot-service/internal/rabbitmq"
	"tgbot/bot-service/internal/services"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func connectionWithTelegram() *tgbotapi.BotAPI {
	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		logrus.Fatal("TELEGRAM_TOKEN environment variable is not set")
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		logrus.Fatal("Cannot connect to telegram")
	}
	logrus.Info("Connected to Telegram successfully")
	return bot
}

func startGRPCServer(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	// add a listener address
	lis, err := net.Listen("tcp", ":50001")
	if err != nil {
		logrus.Fatalf("ERROR STARTING THE SERVER : %v", err)
	}

	// start the grpc server
	grpcServer := grpc.NewServer()

	// // Register health service
	// healthServer := health.NewServer()
	// healthServer.SetServingStatus("task.TaskService", grpc_health_v1.HealthCheckResponse_SERVING)
	// grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	// start serving to the address
	go func() {
		if err := grpcServer.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			logrus.Fatalf("grpcServer can't serve: %v", err)
			return
		}
	}()

	select {
	case <-ctx.Done():
		grpcServer.GracefulStop()
		return
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mainWG := sync.WaitGroup{}

	mainWG.Add(1)
	go startGRPCServer(ctx, &mainWG)

	logrus.SetFormatter(new(logrus.JSONFormatter))
	semaphor := make(chan struct{}, 1000)

	//Start bot
	bot := connectionWithTelegram()
	sessionStorage := services.CreateSessionStorage(ctx)

	//Get config message chan
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60
	updateConfig.AllowedUpdates = []string{"message", "callback_query"}

	//Init rabbitMQ
	r := rabbitmq.NewRabbitMQ("amqp://guest:guest@localhost:5672/")
	defer r.Close()

	r.DeclareQueue(rabbitmq.DelayedExchange, rabbitmq.NotifyTaskQueue, rabbitmq.NotifyKey)
	msgs := r.ConsumeChan(rabbitmq.NotifyTaskQueue)

	// Start consume messages
	notifyWorkersPool(2, ctx, msgs, bot)

	//handle graceful shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		logrus.Infof("Received shutdown signal: %v", sig)
		cancel()
		bot.StopReceivingUpdates()
	}()

	logrus.Info("Service bot-service started")
	//Work with message chan
	for update := range bot.GetUpdatesChan(updateConfig) {
		semaphor <- struct{}{}
		mainWG.Add(1)
		go handlers.HandlUpdate(bot, update, &mainWG, semaphor, sessionStorage)
	}

	mainWG.Wait()
}

func notifyWorkersPool(countWorkers int, ctx context.Context, msgs <-chan amqp091.Delivery, bot *tgbotapi.BotAPI) {
	for i := 0; i < countWorkers; i++ {
		go func(ctx context.Context, i int, bot *tgbotapi.BotAPI) {
			for {
				select {
				case <-ctx.Done():
					logrus.Infof("notifyWorkersPool, Worker №%d end work", i)
					return
				default:
					for mes := range msgs {
						var notifyTask rabbitmq.TaskNotify
						if err := json.Unmarshal(mes.Body, &notifyTask); err != nil {
							logrus.Errorf("notifyWorkersPool, can`t unmarshal task, err%v", err)
							continue
						}

						msg := tgbotapi.NewMessage(int64(notifyTask.ChatID), fmt.Sprintf("Напоминание %s\n%s", notifyTask.Name, notifyTask.Description))
						if _, err := bot.Send(msg); err != nil {
							logrus.Errorf("failed to send telegram msg: %v", err)
						}
					}
				}
			}
		}(ctx, i+1, bot)
	}
}
