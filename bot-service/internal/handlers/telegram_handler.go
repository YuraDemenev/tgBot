package handlers

import (
	"fmt"
	"strings"
	"sync"

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

func addTask(bot *tgbotapi.BotAPI, update *tgbotapi.Update, chatId int64) {

}

func HandlUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update, mainWg *sync.WaitGroup, symahor chan struct{}) {
	defer func() {
		<-symahor
	}()
	defer mainWg.Done()

	if update.Message == nil {
		return
	}

	text := update.Message.Text
	chatId := update.Message.Chat.ID
	userName := update.Message.Chat.UserName

	switch text {
	case "/start":
		str := fmt.Sprintf("Привет %s", userName)
		if err := sendMessage(bot, str, chatId); err != nil {
			//TODO добавить обработку
		}

	case "/addTask":
		str := fmt.Sprintf(`%s чтобы добавить задачу опиште вашу задачу в формате: 
		Имя задачи, Описание, дата, время. 
		Пример:\n Поход к врачу, Сегодня в 15:00 запись к зубному, 25.08.2025, 12:00`, userName)
		if err := sendMessage(bot, str, chatId); err != nil {
			//TODO добавить обработку
		}

	case "/deleteTask":
	case "/changeTask":

	case "/myTasks":

	default:
		str := fmt.Sprintf(`Извини %s, но я тебя не понимаю давай попробуем ещё раз. Напиши комманду которую я знаюю=`, userName)
		if err := sendMessage(bot, str, chatId); err != nil {
			//TODO добавить обработку
		}
	}
}
