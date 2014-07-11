package tests

import (
	"bitbucket.org/daemonfire300/pleyus-alpha/app/models"
	"database/sql"
	"fmt"
	"github.com/coopernurse/gorp"
	_ "github.com/lib/pq"
	"github.com/revel/revel"
	"net/url"
)

var (
	Dbm *gorp.DbMap
	db  *sql.DB
)

type AppTest struct {
	revel.TestSuite
}

func initDB() {
	var err error
	db, err = sql.Open("postgres", "user=postgres dbname=pleyus_test password=abc sslmode=disable")
	if err != nil {
		revel.ERROR.Fatal("Could not establish database connection")
	} else {
		revel.INFO.Println("Connected to the database")
	}
	Dbm = &gorp.DbMap{Db: db, Dialect: gorp.PostgresDialect{}}
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
	Dbm.TraceOn("[===|gorp-test-suite|===]", revel.INFO)
	err := Dbm.CreateTablesIfNotExists()
	if err != nil {
		revel.ERROR.Fatal("Could not create tables")
	} else {
		revel.INFO.Println("Dbm.CreateTablesIfNotExists() ran without any errors")
	}
}

func tearDown() {
	Dbm.DropTablesIfExists()
}

func createTestUser() *models.User {
	u := &models.User{
		Username: "TestUserA",
		Email:    "test@accountr.eu",
		Password: "stronk_hidden_password",
	}
	return u
}

func (t *AppTest) Before() {
	println("Set up")
	initDB()
	setupTables()
}

func (t *AppTest) userShouldBeCreated(u *models.User) bool {
	txn, err := Dbm.Begin()
	if err != nil {
		panic(err)
	}
	err = txn.SelectOne(u, "SELECT * FROM user WHERE username = $1 AND email = $2", u.Username, u.Email)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func (t *AppTest) TestRegisterUser() {
	u := createTestUser()
	d := url.Values{}
	d.Add("user.Username", u.Username)
	d.Add("user.Email", u.Email)
	d.Add("user.Password", u.Password)
	t.PostForm("/register", d)
	t.AssertStatus(200)
	t.AssertOk()
	t.Assert(t.userShouldBeCreated(u))
}

func (t *AppTest) After() {
	println("Tear down")
	tearDown()
}
