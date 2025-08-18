package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"tgbot/bot-service/internal/handlers"
	"tgbot/bot-service/internal/services"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
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

func startGRPCServer(ctx context.Context) {
	// add a listener address
	lis, err := net.Listen("tcp", ":50001")
	if err != nil {
		logrus.Fatalf("ERROR STARTING THE SERVER : %v", err)
	}

	// start the grpc server
	grpcServer := grpc.NewServer()

	// Register health service
	healthServer := health.NewServer()
	healthServer.SetServingStatus("task.TaskService", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go startGRPCServer(ctx)

	//Start bot
	logrus.SetFormatter(new(logrus.JSONFormatter))
	mainWG := sync.WaitGroup{}
	semaphor := make(chan struct{}, 1000)

	bot := connectionWithTelegram()
	sessionStorage := services.CreateSessionStorage()

	//Get config message chan
	updateConfig := tgbotapi.NewUpdate(0)

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
