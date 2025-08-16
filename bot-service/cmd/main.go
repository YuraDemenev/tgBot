package main

import (
	"os"
	"sync"
	"tgbot/bot-service/internal/handlers"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
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
	logrus.SetFormatter(new(logrus.JSONFormatter))
	mainWG := sync.WaitGroup{}
	symahor := make(chan struct{}, 1000)

	bot := connectionWithTelegram()
	//Get config message chan
	updateConfig := tgbotapi.NewUpdate(0)
	//Work with message chan
	for update := range bot.GetUpdatesChan(updateConfig) {
		symahor <- struct{}{}
		mainWG.Add(1)
		go handlers.HandlUpdate(bot, update, &mainWG, symahor)
	}

	mainWG.Wait()
}
