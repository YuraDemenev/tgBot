package handlers

import (
	"fmt"
	"strings"
	"sync"

	"tgbot/bot-service/internal/services"
	"tgbot/bot-service/internal/states"

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

func HandlUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update, mainWg *sync.WaitGroup,
	semaphor chan struct{}, sessionStorage *services.SessionStorage) {
	defer func() {
		<-semaphor
	}()
	defer mainWg.Done()

	if update.Message == nil {
		return
	}

	text := update.Message.Text
	chatId := update.Message.Chat.ID
	userName := update.Message.Chat.UserName

	status := sessionStorage.GetStatus(userName)
	if status == states.GetZeroValue() {
		handleCommands(bot, chatId, text, userName, sessionStorage)
	} else {
		handleStates(bot, status, sessionStorage, text, userName, chatId)
	}
}

func handleCommands(bot *tgbotapi.BotAPI, chatId int64, text, userName string, sessionStorage *services.SessionStorage) {
	switch text {
	case "/start":
		str := fmt.Sprintf("Привет %s", userName)
		if err := sendMessage(bot, str, chatId); err != nil {
			logrus.Errorf("handler commands, /start get error: %v", err)
			return
		}

	case "/addTask":
		str := fmt.Sprintf(`%s чтобы добавить задачу опишите вашу задачу в формате:\n
		Имя задачи, Описание, дата, время.
		Пример:\n Поход к врачу, Сегодня в 15:00 запись к зубному, 25.08.2025, 12:00`, userName)
		if err := sendMessage(bot, str, chatId); err != nil {
			logrus.Errorf("handler commands, /addTask get error: %v", err)
			return
		}
		sessionStorage.StoreSession(userName, states.AddTask)

	case "/deleteTask":
		sessionStorage.StoreSession(userName, states.DeleteTask)
	case "/changeTask":
		sessionStorage.StoreSession(userName, states.ChangeTask)
	case "/myTasks":
		sessionStorage.StoreSession(userName, states.MyTasks)

	default:
		str := fmt.Sprintf(`Извини %s, но я тебя не понимаю давай попробуем ещё раз. Напиши комманду которую я знаю`, userName)
		if err := sendMessage(bot, str, chatId); err != nil {
			//TODO добавить обработку
		}
	}
}

func handleStates(bot *tgbotapi.BotAPI, status states.Status, sessionStorage *services.SessionStorage, text, userName string, chatId int64) {
	switch status {
	case states.AddTask:
		err := SendTaskGRPC(text)
		if err != nil {
			//Write message to user
			str := fmt.Sprintf(`Извини %s, но кажется ты совершил ошибку, давай попробуем ещё раз. Напиши сообщение для выполнения комманды`, userName)
			if err := sendMessage(bot, str, chatId); err != nil {
				//TODO добавить обработку
			}
			logrus.Errorf("handleStates, AddTask get error: %v", err)
			return
		}
	case states.DeleteTask:
	case states.ChangeTask:
	case states.MyTasks:
	default:
		logrus.Errorf("Uknown status: %v", status)
		//Set zero value
		sessionStorage.StoreSession(userName, states.GetZeroValue())
	}
}
