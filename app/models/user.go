package models

import (
	"code.google.com/p/go.crypto/bcrypt"
	"github.com/coopernurse/gorp"
	"github.com/revel/revel"
	"time"
)

type User struct {
	Id        int64
	Username  string
	Email     string
	Password  string
	Salt      string
	LastLogin time.Time
	Created   time.Time
	Lobby     *Lobby
	LobbyId   int64
}

func NewUser(id int64, username string, email string, password string, salt string, lobby *Lobby) *User {
	return &User{
		Id:       id,
		Username: username,
		Email:    email,
		Password: password,
		Salt:     salt,
		Lobby:    lobby,
	}
}

func (u *User) HashPassword() {
	hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.MinCost)
	if err != nil {
		revel.INFO.Println(err)
	}
	u.Password = string(hash[:])
}

func (u *User) Validate(v *revel.Validation) {
	user := u // only for mapping reason
	v.Required(user.Password).Message("Password required")
	v.MinSize(user.Password, 5).Message("Use more than 5 characters")
	v.MaxSize(user.Password, 128).Message("Use less than 128 characters")
	v.Required(user.Username).Message("Username required")
	v.Check(user.Email, revel.Required{})
	v.Email(user.Email).Message("Not a valid Email")
}

func (u *User) ValidatePassword(v *revel.Validation, password string) {
	v.Required(password)
	v.MinSize(password, 5)
	v.MaxSize(password, 128)
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	v.Required(err == nil)
}

func (u *User) IsOwner() bool {
	if u.Lobby != nil {
		return (u.Lobby.OwnerId == u.Id)
	} else {
		return false
	}
}

func (u *User) GetLobby(txn *gorp.Transaction) {
	var (
		obj interface{}
		err error
	)

	if u.Lobby == nil && u.LobbyId > 0 {
		obj, err = txn.Get(Lobby{}, u.LobbyId)
		if err != nil {
			revel.INFO.Println(err)
		}
		if obj != nil {
			u.Lobby = obj.(*Lobby)
		}
	}
}

func (u *User) HasLobby(txn *gorp.Transaction) bool {
	u.GetLobby(txn)
	return (u.Lobby != nil)
}

func (u *User) PreInsert(_ gorp.SqlExecutor) error {
	if u.Lobby != nil {
		u.LobbyId = u.Lobby.Id
	}
	return nil
}

func (u *User) PreUpdate(_ gorp.SqlExecutor) error {
	if u.Lobby != nil {
		u.LobbyId = u.Lobby.Id
	}
	return nil
}

/*func (u *User) PostGet(exe gorp.SqlExecutor) error {
	var (
		obj interface{}
		err error
	)
	//if u.LobbyId > 1 {
	if u.Lobby == nil {
		obj, err = exe.Get(Lobby{}, u.LobbyId)
		if err != nil {
			revel.INFO.Println(err)
			return err
		}
		if obj != nil {
			u.Lobby = obj.(*Lobby)
		}
	}
	//}
	return nil
}*/
