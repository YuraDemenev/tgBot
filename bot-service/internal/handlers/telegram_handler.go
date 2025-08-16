package handlers

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

func sendMessage(bot *tgbotapi.BotAPI, msg string, chatId int64) error {
	msgConfig := tgbotapi.NewMessage(chatId, msg)
	_, err := bot.Send(msgConfig)
	if err != nil {
		logrus.Errorf("Error sending message: %v", err)
		return err
	}
	return nil
}

func isMessageForBot(update *tgbotapi.Update) bool {
	if update.Message.Text == "" || strings.Contains(update.Message.Text, "/") {
		return false
	}
	return true
}

func sendAnswer(bot *tgbotapi.BotAPI, update *tgbotapi.Update, chatId int64) error {
	msg := tgbotapi.NewMessage(chatId, "answer")
	msg.ReplyToMessageID = update.Message.MessageID
	_, err := bot.Send(msg)
	if err != nil {
		logrus.Errorf("Error sending answer: %v", err)
		return err
	}
	return nil
}

func MainHandler(bot *tgbotapi.BotAPI, updateConfig tgbotapi.UpdateConfig) {
	for update := range bot.GetUpdatesChan(updateConfig) {
		if update.Message == nil {
			continue
		}

		text := update.Message.Text

		switch text {
		case "/start":
			if err := sendMessage(bot, "Привет test", update.Message.Chat.ID); err != nil {
				//TODO добавить обработку
			}

		default:
			if isMessageForBot(&update) {
				if err := sendAnswer(bot, &update, update.Message.Chat.ID); err != nil {
					//TODO добавить обработку
				}
			}
		}
	}
}
