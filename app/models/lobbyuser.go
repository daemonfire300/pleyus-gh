package models

import (
	"github.com/coopernurse/gorp"
)

type LobbyUser struct {
	Id      int64
	UserId  int64
	LobbyId int64
	Rated   bool
	Active  bool
	Lobby   *Lobby
	User    *User
}

func (l *LobbyUser) PreInsert(_ gorp.SqlExecutor) error {
	if l.User != nil {
		l.UserId = l.User.Id
	}
	if l.Lobby != nil {
		l.LobbyId = l.Lobby.Id
	}
	return nil
}

func (l *LobbyUser) PreUpdate(_ gorp.SqlExecutor) error {
	if l.User != nil {
		l.UserId = l.User.Id
	}
	if l.Lobby != nil {
		l.LobbyId = l.Lobby.Id
	}
	return nil
}
