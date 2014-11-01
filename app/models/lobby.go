package models

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"code.google.com/p/go.crypto/bcrypt"
	"github.com/coopernurse/gorp"
	"github.com/revel/revel"
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
	Players            map[int64]*User
	Rating             int
}

type LobbyMeta struct {
	Id              int64
	LobbyId         int64
	Lobby           *Lobby
	Description     string
	Server          string
	ServerType      string
	VoiceServer     string
	VoiceServerType string
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

func (l *Lobby) Started() bool {
	return l.State == "started"
}

func (l *Lobby) Ended() bool {
	return l.State == "done"
}

func (lobby *Lobby) Validate(v *revel.Validation) {
	v.Required(lobby.SkillLevel)
	v.MinSize(lobby.Title, 4)
	v.MaxSize(lobby.Title, 128)
	v.Required(lobby.Title)
	v.Required(lobby.GameId)
}

func (l *Lobby) IsValidRating(r int) bool {
	return revel.ValidRange(0, 10).IsSatisfied(r)
}

func (l *Lobby) PreInsert(_ gorp.SqlExecutor) error {
	if l.Owner != nil {
		l.OwnerId = l.Owner.Id
	}
	if l.Game != nil {
		l.GameId = l.Game.Id
	}
	l.Updated = time.Now()
	l.Created = time.Now()
	return nil
}

func (l *Lobby) PreUpdate(_ gorp.SqlExecutor) error {
	if l.Owner != nil {
		l.OwnerId = l.Owner.Id
	}
	if l.Game != nil {
		l.GameId = l.Game.Id
	}
	l.Updated = time.Now()
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
	plrs, err := txn.Select(User{}, "SELECT u.* FROM userlobby ul INNER JOIN users u ON u.id = ul.userid WHERE ul.active = $1 AND ul.lobbyid = $2", true, l.Id)
	l.Players = make(map[int64]*User)
	if err != nil {
		revel.INFO.Println(err)
	} else {
		for _, r := range plrs {
			plr := r.(*User)
			l.Players[plr.Id] = plr
		}
	}
}

// NOTE: This method IS NOT state-sensitive, this means, if the lobby has been closed all players
// have been rated etc, this will still return true if the lobby has not been cleared yet.
// NOTE: This method DOES NOT check if the lobby exists.
func (l *Lobby) HasPlayer(txn *gorp.Transaction, p *User) (r bool) {
	h, err := txn.SelectStr("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1 AND lobbyid = $2)", p.Id, l.Id)
	if err != nil {
		revel.INFO.Println(err)
	}
	r, _ = strconv.ParseBool(h)
	return
}

func (l *Lobby) GetMeta(txn *gorp.Transaction) (*LobbyMeta, error) {
	var m LobbyMeta
	err := txn.SelectOne(&m, "SELECT * FROM lobbymeta WHERE lobbyid = $1", l.Id)
	if err != nil && err != sql.ErrNoRows {
		revel.ERROR.Println(err)
		panic(err)
	}
	if err != nil {
		revel.INFO.Println(err)
		return nil, err
	} else {
		m.Lobby = l
		return &m, err
	}
}
