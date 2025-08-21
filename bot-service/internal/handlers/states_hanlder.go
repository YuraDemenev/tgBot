package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"tgbot/bot-service/internal/services"
	"tgbot/bot-service/internal/states"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

func AddTaskHandler(text, userName string, bot *tgbotapi.BotAPI, chatID int64, sessionStorage *services.SessionStorage) {
	logrus.Infof("user: %s, started addTask", userName)

	err := SendTaskGRPC(text, userName)
	if err != nil {
		//Write message to user
		str := fmt.Sprintf(`Извини %s, но кажется ты совершил ошибку, давай попробуем ещё раз. Напиши сообщение для выполнения комманды`, userName)
		if err := sendMessage(bot, str, chatID, userName); err != nil {
			logrus.Errorf("handlerStates, can`t send message, error: %v", err)
			return
		}
		logrus.Errorf("handleStates, AddTask get error: %v", err)
		return
	}
	//write message to user
	str := fmt.Sprint(`Твоя задача была успешно сохранена, я пришлю тебе уведомление в назначенное время`)
	if err := sendMessage(bot, str, chatID, userName); err != nil {
		logrus.Errorf("handlerStates, can`t send message, error: %v", err)
		return
	}
	sessionStorage.StoreSession(userName, states.GetDefaultValue())
	return
}

func DeleteTaskHandler(text, userName string, bot *tgbotapi.BotAPI, chatID int64, sessionStorage *services.SessionStorage) {
	logrus.Infof("user: %s, started DeleteTask", userName)
	num, err := strconv.Atoi(text)
	if err != nil {
		str := fmt.Sprintf(`Извини %s, но кажется ты написал не номер, давай попробуем ещё раз. 
			Напиши номер задачи (например 6), которую ты хочешь удалить`, userName)
		if err := sendMessage(bot, str, chatID, userName); err != nil {
			logrus.Errorf("handlerStates, can`t send message, error: %v", err)
			return
		}
		logrus.Errorf("handleStates, AddTask get error: %v", err)
		return
	}

	err = DeleteUserTasks(userName, num)
	if err != nil {
		//Write message to user
		str := fmt.Sprintf(`Извини %s, но кажется ты написал не номер, давай попробуем ещё раз. 
			Напиши номер задачи (например 6), которую ты хочешь удалить`)
		if err := sendMessage(bot, str, chatID, userName); err != nil {
			logrus.Errorf("handlerStates, can`t send message, error: %v", err)
			return
		}
		logrus.Errorf("handleStates, AddTask get error: %v", err)
		return
	}

	str := fmt.Sprintf(`Сообщение под номером %d было успешно удалено`, num)
	if err := sendMessage(bot, str, chatID, userName); err != nil {
		logrus.Errorf("handlerStates, can`t send message, error: %v", err)
		return
	}

	sessionStorage.StoreSession(userName, states.GetDefaultValue())
}

func ChangeTaskHandler(text, userName string, bot *tgbotapi.BotAPI, chatID int64,
	sessionStorage *services.SessionStorage, update tgbotapi.Update) {
	logrus.Infof("user: %s, started ChangeTask", userName)

	if update.CallbackQuery != nil && update.Message == nil {
		str := fmt.Sprintf(`%s напиши номер задачи, которую ты хочешь поменять (просто цифрой, например 6) и новое значение.
		Пример: 2, Вечером забрать посылку с почты`, userName)
		if err := sendMessage(bot, str, chatID, userName); err != nil {
			logrus.Errorf("handler commands, /ChangeTaskHandler can`t send message, error: %v", err)
			return
		}
		sessionStorage.SetMetaData(userName, update.CallbackQuery.Data)
		return
	}
	if update.Message != nil {
		if update.CallbackQuery != nil {
			logrus.Info(update.CallbackQuery.Data)
		}

		// In arr save task num, new value
		test := update.Message.Text
		arr := strings.Split(test, ",")
		if len(arr) != 2 {
			str := fmt.Sprintf(`%s похоже, что ты неправильно написал сообщение, напиши его ещё раз.
			Пример: 2, Вечером забрать посылку с почты`, userName)
			if err := sendMessage(bot, str, chatID, userName); err != nil {
				logrus.Errorf("handler commands, /ChangeTaskHandler can`t send message, error: %v", err)
				return
			}
			return
		}
		for i, v := range arr {
			arr[i] = strings.TrimSpace(v)
		}

		// After this part, after err or success result sessionStorage has to get defaultValue
		defer sessionStorage.StoreSession(userName, states.GetDefaultValue())

		//Get meta data from storage
		changeValueMeta := sessionStorage.GetMetaData(userName)
		if changeValueMeta == nil {
			err := fmt.Errorf("handler commands, /ChangeTaskHandler can`t get meta change value from sessionStorage")
			logrus.Errorf(err.Error())
			str := fmt.Sprintf(`%s похоже, что что-то пошло не так, давай попробуем ещё раз, напиши мне команду`, userName)
			if err := sendMessage(bot, str, chatID, userName); err != nil {
				logrus.Errorf("handler commands, /ChangeTaskHandler can`t send message, error: %v", err)
				return
			}
			return
		}

		//Convert meta data to string
		changeValue, ok := changeValueMeta.(string)
		if ok != true {
			logrus.Error("handler commands, /ChangeTaskHandler can`t convert changeValueMeta to str")
			str := fmt.Sprintf(`%s похоже, что что-то пошло не так, давай попробуем ещё раз, напиши мне команду`, userName)
			if err := sendMessage(bot, str, chatID, userName); err != nil {
				logrus.Errorf("handler commands, /ChangeTaskHandler can`t send message, error: %v", err)
				return
			}
			return
		}

		//Get taskNum
		taskNum, err := strconv.Atoi(arr[0])
		if err != nil {
			logrus.Errorf("handler commands, /ChangeTaskHandler can`t convert changeValueMeta to str, err:%v", err)
			str := fmt.Sprintf(`%s похоже, что что-то пошло не так, давай попробуем ещё раз, напиши мне команду`, userName)
			if err := sendMessage(bot, str, chatID, userName); err != nil {
				logrus.Errorf("handler commands, /ChangeTaskHandler can`t send message, error: %v", err)
				return
			}
			return
		}

		err = ChangeTask(userName, arr[1], changeValue, taskNum)
		if err != nil {
			logrus.Errorf("handler commands, /ChangeTaskHandler can`t change task, err%v", err)
			return
		}
	}
}
