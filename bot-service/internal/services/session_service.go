package services

import "time"

type session struct {
	timeOut       time.Time
	sessionAction string
}
