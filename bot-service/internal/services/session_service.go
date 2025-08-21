package services

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"tgbot/bot-service/internal/states"
	"time"

	"github.com/sirupsen/logrus"
)

type session struct {
	timeOut       time.Time
	sessionAction states.Status
	metaData      interface{}
}

type SessionStorage struct {
	hashTable sync.Map
}

func (s *SessionStorage) SetMetaData(userName string, metaData interface{}) {
	curStatus := s.GetStatus(userName)
	session := session{timeOut: time.Now().Add(time.Minute * 10), sessionAction: curStatus, metaData: metaData}
	s.hashTable.Store(getUserHash(userName), session)
}

func (s *SessionStorage) GetMetaData(userName string) interface{} {
	value, ok := s.hashTable.Load(getUserHash(userName))
	if !ok {
		logrus.Error(fmt.Errorf("can`t get value from redis"))
		return nil
	}
	session, ok := value.(session)
	if !ok {
		logrus.Errorf("Session Service, can`t convert value: %v to sesion", value)
		return states.GetDefaultValue()
	}
	metaData := session.metaData
	return metaData
}

func (s *SessionStorage) GetStatus(userName string) states.Status {
	userHash := getUserHash(userName)
	value, ok := s.hashTable.Load(userHash)
	if !ok {
		logrus.Info(fmt.Errorf("can`t get value from redis"))
		return states.GetDefaultValue()
	}

	session, ok := value.(session)
	if !ok {
		logrus.Errorf("Session Service, can`t convert value: %v to sesion", value)
		return states.GetDefaultValue()
	}
	status := session.sessionAction

	return status
}

func (s *SessionStorage) StoreSession(userName string, status states.Status) {
	userHash := getUserHash(userName)
	session := session{timeOut: time.Now().Add(time.Minute * 10), sessionAction: status}
	s.hashTable.Store(userHash, session)
}

func CreateSessionStorage() *SessionStorage {
	sessionStorage := SessionStorage{hashTable: sync.Map{}}
	return &sessionStorage
}

func getUserHash(userName string) string {
	salt := "asd123kgpfoa"

	h := sha256.New()
	h.Write([]byte(userName))

	return hex.EncodeToString(h.Sum([]byte(salt)))
}

//TODO worker pool for clear storage
