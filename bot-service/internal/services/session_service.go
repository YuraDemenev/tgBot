package services

import (
	"context"
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

func CreateSessionStorage(ctx context.Context) *SessionStorage {
	sessionStorage := SessionStorage{hashTable: sync.Map{}}
	startWorkerPool(ctx, 2, &sessionStorage)
	return &sessionStorage
}

func getUserHash(userName string) string {
	salt := "asd123kgpfoa"

	h := sha256.New()
	h.Write([]byte(userName))

	return hex.EncodeToString(h.Sum([]byte(salt)))
}

func (s *SessionStorage) DeleteSession(userName string) {
	s.hashTable.Delete(getUserHash(userName))
}

func startWorkerPool(ctx context.Context, countWorkers int, sessionStorage *SessionStorage) {
	for i := 0; i < countWorkers; i++ {
		go func(ctx context.Context, sessionStorage *SessionStorage) {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					sessionStorage.hashTable.Range(func(key, value any) bool {
						session, ok := value.(session)
						if !ok {
							logrus.Errorf("workers pool, can`t convert value to session")
							return false
						}
						if session.timeOut.Before(time.Now()) {
							sessionStorage.DeleteSession(key.(string))
						}

						return true
					})
					time.Sleep(time.Second * 5)
				}
			}
		}(ctx, sessionStorage)
	}
}
