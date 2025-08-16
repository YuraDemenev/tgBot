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

func (s *SessionStorage) GetStatus(userHash string) states.Status {
	value, ok := s.hashTable.Load(userHash)
	if !ok {
		return states.GetZeroValue()
	}

	status, ok := value.(states.Status)
	if !ok {
		logrus.Errorf("Session Service, can`t convert value: %v to status", value)
		return states.GetZeroValue()
	}

	return status
}

func (s *SessionStorage) StoreSession(userHash string, status states.Status) {
	session := session{timeOut: time.Now().Add(time.Minute * 10), sessionAction: status}
	s.hashTable.Store(userHash, session)
}

func CreateSessionStorage() *SessionStorage {
	sessionStorage := SessionStorage{hashTable: sync.Map{}}
	return &sessionStorage
}

func GetUserHash(userName string) string {
	salt := "asd123kgpfoa"

	h := sha256.New()
	h.Write([]byte(userName))

	return hex.EncodeToString(h.Sum([]byte(salt)))
}
