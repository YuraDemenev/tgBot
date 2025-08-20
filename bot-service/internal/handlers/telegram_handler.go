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

func sendMessage(bot *tgbotapi.BotAPI, msg string, chatId int64, userName string) error {
	logrus.Infof("send message to user: %s", userName)

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
	if status == states.GetDefaultValue() {
		handleCommands(bot, chatId, text, userName, sessionStorage)
	} else {
		handleStates(bot, status, sessionStorage, text, userName, chatId)
	}
}

func handleCommands(bot *tgbotapi.BotAPI, chatId int64, text, userName string, sessionStorage *services.SessionStorage) {
	switch text {
	case "/start":
		logrus.Infof("user: %s, started /start", userName)
		str := fmt.Sprintf("Привет %s", userName)

		if err := sendMessage(bot, str, chatId, userName); err != nil {
			logrus.Errorf("handler commands, /start can`t send message, error: %v", err)
			return
		}

	case "/addTask":
		logrus.Infof("user: %s, started /addTask", userName)
		str := fmt.Sprintf(`%s чтобы добавить задачу опишите вашу задачу в формате:
		Имя задачи, Описание, дата, время уведомления.
		Пример:\n Поход к врачу, Сегодня в 15:00 запись к зубному, 25.08.2025, 12:00`, userName)

		if err := sendMessage(bot, str, chatId, userName); err != nil {
			logrus.Errorf("handler commands, /addTask can`t send message, error: %v", err)
			return
		}

		sessionStorage.StoreSession(userName, states.AddTask)

	case "/deleteTask":
		logrus.Infof("user: %s, started /deleteTask", userName)
		sessionStorage.StoreSession(userName, states.DeleteTask)

	case "/changeTask":
		logrus.Infof("user: %s, started /changeTask", userName)
		sessionStorage.StoreSession(userName, states.ChangeTask)

	case "/myTasks":
		logrus.Infof("user: %s, started /myTask", userName)
		sessionStorage.StoreSession(userName, states.MyTasks)
		tasks, err := GetUserTasks(userName)
		if err != nil {
			logrus.Errorf("user %s, did`t get tasks", userName)
			return
		}
		logrus.Info("user %s, got his task", userName)
		var builder strings.Builder
		for i, v := range tasks {
			builder.Write([]byte(fmt.Sprintf("Задача №%d\n", i+1)))
			builder.Write([]byte(v.Name + "\n"))
			builder.Write([]byte(v.Description + "\n"))
			builder.Write([]byte(fmt.Sprintf("Дата: %02d.%02d.%04d\n", v.Date.Day, v.Date.Month, v.Date.Year)))
			builder.Write([]byte("время: " + v.Time.AsTime().Format("15:04") + "\n\n"))
		}

		if err := sendMessage(bot, builder.String(), chatId, userName); err != nil {
			logrus.Errorf("handler commands, /myTasks can`t send message, error: %v", err)
			return
		}

		sessionStorage.StoreSession(userName, states.GetDefaultValue())
	default:
		str := fmt.Sprintf(`Извини %s, но я тебя не понимаю давай попробуем ещё раз. Напиши комманду которую я знаю`, userName)
		if err := sendMessage(bot, str, chatId, userName); err != nil {
			logrus.Errorf("handler commands, default can`t send message, error: %v", err)
			return
		}
	}
}

func handleStates(bot *tgbotapi.BotAPI, status states.Status, sessionStorage *services.SessionStorage, text, userName string, chatId int64) {
	switch status {
	case states.AddTask:
		logrus.Infof("user: %s, started addTask", userName)

		err := SendTaskGRPC(text, userName)
		if err != nil {
			//Write message to user
			str := fmt.Sprintf(`Извини %s, но кажется ты совершил ошибку, давай попробуем ещё раз. Напиши сообщение для выполнения комманды`, userName)
			if err := sendMessage(bot, str, chatId, userName); err != nil {
				logrus.Errorf("handlerStates, can`t send message, error: %v", err)
				return
			}
			logrus.Errorf("handleStates, AddTask get error: %v", err)
			return
		}
		//write message to user
		str := fmt.Sprint(`Твоя задача была успешно сохранена, я пришлю тебе уведомление в назначенное время`)
		if err := sendMessage(bot, str, chatId, userName); err != nil {
			logrus.Errorf("handlerStates, can`t send message, error: %v", err)
			return
		}
		sessionStorage.StoreSession(userName, states.GetDefaultValue())
		return

	case states.DeleteTask:
		logrus.Infof("user: %s, started DeleteTask", userName)
	case states.ChangeTask:
		logrus.Infof("user: %s, started ChangeTask", userName)
	case states.MyTasks:
		logrus.Infof("user: %s, started MyTask", userName)

	default:
		logrus.Errorf("Uknown status: %v", status)
		//Set zero value
		sessionStorage.StoreSession(userName, states.GetDefaultValue())
	}
}
