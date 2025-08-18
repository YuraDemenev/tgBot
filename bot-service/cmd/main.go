package main

import (
	"net"
	"os"
	"sync"
	"tgbot/bot-service/internal/handlers"
	"tgbot/bot-service/internal/services"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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

func main() {
	// add a listener address
	lis, err := net.Listen("tcp", ":50001")
	if err != nil {
		logrus.Fatalf("ERROR STARTING THE SERVER : %v", err)
	}

	// start the grpc server
	grpcServer := grpc.NewServer()
	// start serving to the address
	logrus.Fatal(grpcServer.Serve(lis))

	//Start bot
	logrus.SetFormatter(new(logrus.JSONFormatter))
	mainWG := sync.WaitGroup{}
	semaphor := make(chan struct{}, 1000)

	bot := connectionWithTelegram()
	sessionStorage := services.CreateSessionStorage()

	//Get config message chan
	updateConfig := tgbotapi.NewUpdate(0)
	//Work with message chan
	for update := range bot.GetUpdatesChan(updateConfig) {
		semaphor <- struct{}{}
		mainWG.Add(1)
		go handlers.HandlUpdate(bot, update, &mainWG, semaphor, sessionStorage)
	}

	mainWG.Wait()
}
