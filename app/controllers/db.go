package controllers

import (
	"database/sql"
	"github.com/coopernurse/gorp"
	//"github.com/daemonfire300/pleyusweb/app/models"
	"bitbucket.org/daemonfire300/pleyus-alpha/app/models"
	_ "github.com/lib/pq"
	"github.com/revel/revel"
)

var (
	Dbm *gorp.DbMap
	db  *sql.DB
)

type DatabaseController struct {
	*revel.Controller
	Txn *gorp.Transaction
}

func InitDB() {
	var err error
	db, err = sql.Open("postgres", "user=postgres dbname=pleyus password=abc sslmode=disable")
	if err != nil {
		revel.ERROR.Fatal("Could not establish database connection")
	} else {
		revel.INFO.Println("Connected to the database")
	}
	Dbm = &gorp.DbMap{Db: db, Dialect: gorp.PostgresDialect{}}
	setupTables()
}

func setupTables() {
	userTable := Dbm.AddTableWithName(models.User{}, "users").SetKeys(true, "Id")
	lobbyTable := Dbm.AddTableWithName(models.Lobby{}, "lobbys").SetKeys(true, "Id")
	lobbyuserTable := Dbm.AddTableWithName(models.LobbyUser{}, "userlobby").SetKeys(true, "Id")
	Dbm.AddTableWithName(models.Game{}, "games").SetKeys(true, "Id")
	Dbm.AddTableWithName(models.LobbyInvite{}, "lobbyinvites").SetKeys(true, "Id")
	Dbm.AddTableWithName(models.Friend{}, "friends").SetKeys(true, "Id")
	lobbymetaTable := Dbm.AddTableWithName(models.LobbyMeta{}, "lobbymeta").SetKeys(true, "Id")

	userTable.ColMap("Lobby").Transient = true
	lobbyTable.ColMap("Owner").Transient = true
	lobbyTable.ColMap("Game").Transient = true
	lobbyTable.ColMap("Players").Transient = true
	lobbyuserTable.ColMap("Lobby").Transient = true
	lobbyuserTable.ColMap("User").Transient = true
	lobbymetaTable.ColMap("Lobby").Transient = true
	Dbm.TraceOn("[===|gorp|===]", revel.INFO)
	err := Dbm.CreateTablesIfNotExists()
	if err != nil {
		revel.ERROR.Fatal("Could not create tables")
	} else {
		revel.INFO.Println("Dbm.CreateTablesIfNotExists() ran without any errors")
	}
}

func (c *DatabaseController) Begin() revel.Result {
	txn, err := Dbm.Begin()
	if err != nil {
		panic(err)
	}
	c.Txn = txn
	return nil
}

func (c *DatabaseController) Commit() revel.Result {
	if c.Txn == nil {
		return nil
	}
	if err := c.Txn.Commit(); err != nil && err != sql.ErrTxDone {
		panic(err)
	}
	c.Txn = nil
	return nil
}

func (c *DatabaseController) Rollback() revel.Result {
	if c.Txn == nil {
		return nil
	}
	if err := c.Txn.Rollback(); err != nil && err != sql.ErrTxDone {
		panic(err)
	}
	c.Txn = nil
	return nil
}

func (c *DatabaseController) GetUserById(id int64) (*models.User, error) {
	obj, err := c.Txn.Get(models.User{}, id)
	if err != nil {
		revel.INFO.Println("User with id ", id, " not found")
		return nil, err
	}
	return obj.(*models.User), nil
}

func (c *DatabaseController) GetUserByName(username string) (*models.User, error) {
	var user models.User
	err := c.Txn.SelectOne(&user, "SELECT * FROM users WHERE username = $1", username)
	if err != nil {
		revel.INFO.Println("User with username ", username, " not found")
		revel.INFO.Println(err)
		return nil, err
	}
	revel.INFO.Println(user)
	return &user, nil
}

func (c *DatabaseController) SaveUser(user *models.User) {
	err := c.Txn.Insert(user)
	if err != nil {
		revel.INFO.Println(err)
	}
}

func (c *DatabaseController) UpdateUser(user *models.User) {
	aff, err := c.Txn.Update(user)
	revel.INFO.Println("Affected rows: ", aff)
	if err != nil {
		revel.INFO.Println(err)
	}
}

func (c *DatabaseController) SaveLobby(lobby *models.Lobby) {
	err := c.Txn.Insert(lobby)
	if err != nil {
		revel.INFO.Println(err)
	}
}

func (c *DatabaseController) UpdateLobby(lobby *models.Lobby) {
	aff, err := c.Txn.Update(lobby)
	revel.INFO.Println("Affected rows: ", aff)
	if err != nil {
		revel.INFO.Println(err)
	}
}
