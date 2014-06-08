package models

import (
	"time"
)

type LobbyInvite struct {
	Id       int64
	From     int64
	To       int64
	LobbyId  int64
	Status   string
	Sent     time.Time
	Received time.Time
}
