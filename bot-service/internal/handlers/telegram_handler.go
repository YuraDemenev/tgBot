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

func sendMessage(bot *tgbotapi.BotAPI, msg string, chatID int64, userName string) error {
	logrus.Infof("send message to user: %s", userName)

	msgConfig := tgbotapi.NewMessage(chatID, msg)
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

func sendAnswer(bot *tgbotapi.BotAPI, update *tgbotapi.Update, chatID int64) error {
	msg := tgbotapi.NewMessage(chatID, "answer")
	msg.ReplyToMessageID = update.Message.MessageID
	_, err := bot.Send(msg)
	if err != nil {
		logrus.Errorf("Error sending answer: %v", err)
		return err
	}
	return nil
}

func addTask(bot *tgbotapi.BotAPI, update *tgbotapi.Update, chatID int64) {

}

func HandlUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update, mainWg *sync.WaitGroup,
	semaphor chan struct{}, sessionStorage *services.SessionStorage) {
	defer func() {
		<-semaphor
	}()
	defer mainWg.Done()

	if update.Message == nil && update.CallbackQuery == nil {
		return
	}

	var text string
	var chatID int64
	var userName string
	if update.Message != nil {
		text = update.Message.Text
		chatID = update.Message.Chat.ID
		userName = update.Message.Chat.UserName
	} else {
		text = ""
		chatID = update.CallbackQuery.From.ID
		userName = update.CallbackQuery.From.UserName
	}

	status := sessionStorage.GetStatus(userName)
	if status == states.GetDefaultValue() {
		handleCommands(bot, chatID, text, userName, sessionStorage, update)
	} else {
		handleStates(bot, status, sessionStorage, text, userName, chatID, update)
	}
}

func handleCommands(bot *tgbotapi.BotAPI, chatID int64, text, userName string,
	sessionStorage *services.SessionStorage, update tgbotapi.Update) {
	switch text {
	case "/start":
		logrus.Infof("user: %s, started /start", userName)
		str := fmt.Sprintf("Привет %s", userName)

		if err := sendMessage(bot, str, chatID, userName); err != nil {
			logrus.Errorf("handler commands, /start can`t send message, error: %v", err)
			return
		}

	case "/addTask":
		logrus.Infof("user: %s, started /addTask", userName)
		str := fmt.Sprintf(`%s чтобы добавить задачу опишите вашу задачу в формате:
		Имя задачи, Описание, дата, время уведомления.
		Пример:\n Поход к врачу, Сегодня в 15:00 запись к зубному, 25.08.2025, 12:00`, userName)

		if err := sendMessage(bot, str, chatID, userName); err != nil {
			logrus.Errorf("handler commands, /addTask can`t send message, error: %v", err)
			return
		}

		sessionStorage.StoreSession(userName, states.AddTask)

	case "/deleteTask":
		logrus.Infof("user: %s, started /deleteTask", userName)
		sessionStorage.StoreSession(userName, states.DeleteTask)

		str := fmt.Sprintf(`%s чтобы удалить задачу напиши номер задачи (просто цифрой, например 6), которую хотите удалить. 
		Номер можно получить из списка задач по команде /myTasks`, userName)
		if err := sendMessage(bot, str, chatID, userName); err != nil {
			logrus.Errorf("handler commands, /addTask can`t send message, error: %v", err)
			return
		}

	case "/changeTask":
		logrus.Infof("user: %s, started /changeTask", userName)
		str := fmt.Sprintf(`
		%s выбери поле, которое хочешь поменять`, userName)

		msg := tgbotapi.NewMessage(chatID, str)
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Имя задачи", "Task Name"),
				tgbotapi.NewInlineKeyboardButtonData("Описание", "Description"),
				tgbotapi.NewInlineKeyboardButtonData("Дата", "Date"),
				tgbotapi.NewInlineKeyboardButtonData("Время уведомления", "Time"),
			),
		)

		msg.ReplyMarkup = keyboard
		_, err := bot.Send(msg)
		if err != nil {
			logrus.Errorf("changeTask, can`t send error: %v", err)
		}

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
			builder.Write([]byte("Имя задачи: " + v.Name + "\n"))
			builder.Write([]byte("Описание: " + v.Description + "\n"))
			builder.Write([]byte(fmt.Sprintf("Дата: %02d.%02d.%04d\n", v.Date.Day, v.Date.Month, v.Date.Year)))
			builder.Write([]byte("время: " + v.Time.AsTime().Format("15:04") + "\n\n"))
		}

		if err := sendMessage(bot, builder.String(), chatID, userName); err != nil {
			logrus.Errorf("handler commands, /myTasks can`t send message, error: %v", err)
			return
		}

		sessionStorage.StoreSession(userName, states.GetDefaultValue())
	default:
		str := fmt.Sprintf(`Извини %s, но я тебя не понимаю давай попробуем ещё раз. Напиши комманду которую я знаю`, userName)
		if err := sendMessage(bot, str, chatID, userName); err != nil {
			logrus.Errorf("handler commands, default can`t send message, error: %v", err)
			return
		}
	}
}

func handleStates(bot *tgbotapi.BotAPI, status states.Status, sessionStorage *services.SessionStorage,
	text, userName string, chatID int64, update tgbotapi.Update) {
	switch status {
	case states.AddTask:
		AddTaskHandler(text, userName, bot, chatID, sessionStorage)

	case states.DeleteTask:
		DeleteTaskHandler(text, userName, bot, chatID, sessionStorage)

	case states.ChangeTask:
		ChangeTaskHandler(text, userName, bot, chatID, sessionStorage, update)
	case states.MyTasks:
		logrus.Infof("user: %s, started MyTask", userName)

	default:
		logrus.Errorf("Uknown status: %v", status)
		//Set zero value
		sessionStorage.StoreSession(userName, states.GetDefaultValue())
	}
}
