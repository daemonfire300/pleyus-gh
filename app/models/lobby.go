package models

import (
	"code.google.com/p/go.crypto/bcrypt"
	"database/sql"
	"fmt"
	"github.com/coopernurse/gorp"
	"github.com/revel/revel"
	"time"
)

type Lobby struct {
	Id                 int64
	Title              string
	Created            time.Time
	Updated            time.Time
	Access             string
	Owner              *User
	State              string // open, closed, finished
	Password           sql.NullString
	Game               *Game
	SkillLevel         int
	EstimatedPlayTime  int
	MaxUsers           int
	EstimatedStartTime time.Time
	OwnerId            int64
	GameId             int64
	Players            []*User
	Rating             int
}

func NewLobby(title string, access string, owner *User, state string, game *Game, skillLevel int, estimatedPlayTime int, estimatedStartTime time.Time) *Lobby {
	return &Lobby{
		Title:              title,
		Created:            time.Now(),
		Owner:              owner,
		State:              state,
		Game:               game,
		SkillLevel:         skillLevel,
		EstimatedPlayTime:  estimatedPlayTime,
		EstimatedStartTime: estimatedStartTime,
	}
}

func (l *Lobby) HashPassword() {
	hash, err := bcrypt.GenerateFromPassword([]byte(l.Password.String), bcrypt.MinCost)
	if err != nil {
		revel.INFO.Println(err)
	}
	l.Password.String = string(hash[:])
}

func (l *Lobby) ValidatePassword(v *revel.Validation, password string) {
	v.Required(password)
	v.MinSize(password, 3)
	v.MaxSize(password, 128)
	err := bcrypt.CompareHashAndPassword([]byte(l.Password.String), []byte(password))
	v.Required(err == nil)
}

func (l *Lobby) IsFull() bool {
	return (len(l.Players)+1 > l.MaxUsers)
}

func (lobby *Lobby) Validate(v *revel.Validation) {
	v.Required(lobby.SkillLevel)
	v.MinSize(lobby.Title, 4)
	v.MaxSize(lobby.Title, 128)
	v.Required(lobby.Title)
	v.Required(lobby.GameId)
}

func (l *Lobby) PreInsert(_ gorp.SqlExecutor) error {
	if l.Owner != nil {
		l.OwnerId = l.Owner.Id
	}
	if l.Game != nil {
		l.GameId = l.Game.Id
	}
	return nil
}

func (l *Lobby) PreUpdate(_ gorp.SqlExecutor) error {
	if l.Owner != nil {
		l.OwnerId = l.Owner.Id
	}
	if l.Game != nil {
		l.GameId = l.Game.Id
	}
	return nil
}

func (l *Lobby) PostGet(exe gorp.SqlExecutor) error {
	var (
		obj interface{}
		err error
	)
	/*if l.Owner == nil {
		obj, err = exe.Get(User{}, l.OwnerId)
		if err != nil {
			return fmt.Errorf("Error while fetching related owner(user) %d of lobby %d, error: %s", l.OwnerId, l.Id, err)
		}
		if obj != nil {
			l.Owner = obj.(*User)
		}
	}*/
	if l.Game == nil {
		obj, err = exe.Get(Game{}, l.GameId)
		if err != nil {
			return fmt.Errorf("Error while fetching related game %d of lobby %d, error: %s", l.GameId, l.Id, err)
		}
		if obj != nil {
			l.Game = obj.(*Game)
		}
	}
	return nil
}

func (l *Lobby) GetPlayers(txn *gorp.Transaction) {
	plrs, err := txn.Select(User{}, "SELECT * FROM users WHERE lobbyid = $1", l.Id)
	if err != nil {
		revel.INFO.Println(err)
		l.Players = []*User{}
	} else {
		for _, r := range plrs {
			plr := r.(*User)
			l.Players = append(l.Players, plr)
		}
	}
}
