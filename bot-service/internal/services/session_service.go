package services

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"tgbot/bot-service/internal/states"
	"time"

	"github.com/sirupsen/logrus"
)

type session struct {
	timeOut       time.Time
	sessionAction states.Status
}

type SessionStorage struct {
	hashTable sync.Map
}

func (s *SessionStorage) GetStatus(userName string) states.Status {
	userHash := getUserHash(userName)
	value, ok := s.hashTable.Load(userHash)
	if !ok {
		return states.GetZeroValue()
	}

	session, ok := value.(session)
	status := session.sessionAction
	if !ok {
		logrus.Errorf("Session Service, can`t convert value: %v to sesion", value)
		return states.GetZeroValue()
	}

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
