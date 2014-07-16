package tests

import (
	"bitbucket.org/daemonfire300/pleyus-alpha/app/models"
	"database/sql"
	"fmt"
	"github.com/coopernurse/gorp"
	_ "github.com/lib/pq"
	"github.com/revel/revel"
	//"net/http"
	"net/http/cookiejar"
	"net/url"
)

var (
	Dbm *gorp.DbMap
	db  *sql.DB
)

type AppTest struct {
	revel.TestSuite
	txn *gorp.Transaction
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

func createTestLobby() *models.Lobby {
	l := &models.Lobby{
		Title:             "TestLobbyA",
		Access:            "private",
		SkillLevel:        128,
		EstimatedPlayTime: 60,
		MaxUsers:          32,
	}
	return l
}

func createTestGames() []*models.Game {
	games := []*models.Game{
		&models.Game{
			Name:  "Game A",
			Genre: "FPS",
		},
		&models.Game{
			Name:  "Game B",
			Genre: "Action RPG",
		},
		&models.Game{
			Name:  "Game C",
			Genre: "RPG",
		},
		&models.Game{
			Name:  "Game D",
			Genre: "MMO",
		},
		&models.Game{
			Name:  "Game E",
			Genre: "MMO",
		},
	}
	return games
}

func (t *AppTest) Before() {
	println("Set up")
	initDB()
	setupTables()
	var err error
	t.txn, err = Dbm.Begin()
	if err != nil {
		fmt.Println("panic A")
		panic(err)
	}
}

func (t *AppTest) lobbyShouldBeCreated(l *models.Lobby) bool {
	err := t.txn.SelectOne(l, "SELECT * FROM lobby WHERE title = $1", l.Title)
	if err != nil {
		t.txn.Rollback()
		fmt.Println(err)
		return false
	}
	return true
}

func (t *AppTest) userShouldBeCreated(u *models.User) bool {
	err := t.txn.SelectOne(u, "SELECT * FROM users WHERE username = $1 AND email = $2", u.Username, u.Email)
	if err != nil {
		t.txn.Rollback()
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

func (t *AppTest) TestCreateLobby() {
	u := createTestUser()
	err := t.txn.Insert(u)
	if err != nil {
		t.txn.Rollback()
		panic(err)
		fmt.Println("panic B")
	}
	gs := createTestGames()
	for _, g := range gs {
		err = t.txn.Insert(g)
		if err != nil {
			t.txn.Rollback()
			panic(err)
			fmt.Println("panic C")
		}
	}
	d := url.Values{}
	d.Add("user.Username", u.Username)
	d.Add("user.Password", u.Password)
	j, _ := cookiejar.New(nil)
	t.Client.Jar = j
	fmt.Println("jar error ", err)
	resp, err := t.Client.PostForm("http://localhost:9000/login", d)
	fmt.Println("/login err: ", err)
	resp.Body.Close()

	//fmt.Println("\n\n++++++++COOKIES+++++++", t.Client.Jar, "\n\n")
	//t.Client.Jar.SetCookies(ur, resp.Cookies())
	//fmt.Println("++++++++JAR+++++++", t.Client.Jar)
	if err != nil {
		fmt.Println(err)
		t.Assert(false)
	}
	l := createTestLobby()
	d = url.Values{}
	d.Add("lobby.GameId", "1")
	d.Add("lobby.Title", l.Title)
	d.Add("lobby.Access", "public")
	d.Add("lobby.SkillLevel", "128")
	d.Add("lobby.MaxUsers", "32")
	d.Add("lobby.EstimatedPlayTime", "45")
	d.Add("lobby.EstimatedStartTime", "18:32")
	resp, err = t.Client.PostForm("http://localhost:9000/lobby/create", d)
	resp.Body.Close()
	t.Assert(t.lobbyShouldBeCreated(l))
	t.AssertOk()
	t.AssertStatus(200)
}

func (t *AppTest) After() {
	println("Tear down")
	t.txn.Commit()
	tearDown()
}
