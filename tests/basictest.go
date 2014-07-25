package tests

import (
	"bitbucket.org/daemonfire300/pleyus-alpha/app/models"
	"database/sql"
	"fmt"
	"github.com/coopernurse/gorp"
	_ "github.com/lib/pq"
	"github.com/revel/revel"
	//"net/http"
	//"net/http/cookiejar"
	"net/url"
	"strconv"
)

var (
	Dbm *gorp.DbMap
	db  *sql.DB
	r   int
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

func resetTxn(txn *gorp.Transaction, inErr error) {
	if inErr != nil {
		fmt.Println("rollback .... ", inErr)
		txn.Rollback()
	} else {
		err := txn.Commit()
		if err != nil {
			fmt.Println(err)
			panic(err)
		} else {
			txn, err = Dbm.Begin()
			if err != nil {
				fmt.Println(err)
				panic(err)
			}
		}
	}
}

func createTestUser() *models.User {
	r++
	u := &models.User{
		Username: "TestUserA" + strconv.Itoa(r),
		Email:    "no" + strconv.Itoa(r) + "test@accountr.eu",
		Password: "stronk_hidden_password",
	}
	return u
}

func insertTestUser(txn *gorp.Transaction) (*models.User, string, error) {
	u := createTestUser()
	p := u.Password
	u.HashPassword()
	err := txn.Insert(u)
	return u, p, err
}

func createTestLobby() *models.Lobby {
	r++
	l := &models.Lobby{
		Title:             "TestLobbyA" + strconv.Itoa(r),
		Access:            "private",
		SkillLevel:        128,
		EstimatedPlayTime: 60,
		MaxUsers:          32,
	}
	return l
}

func insertTestLobby(txn *gorp.Transaction, o *models.User) (*models.Lobby, error) {
	l := createTestLobby()
	l.Owner = o
	l.GameId = 3
	err := txn.Insert(l)
	return l, err
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

func insertTestGames(txn *gorp.Transaction) error {
	gs := createTestGames()
	var err error
	for _, g := range gs {
		err = txn.Insert(g)
		if err != nil {
			fmt.Println("panic C")
			fmt.Println(err)
			return err
		}
	}
	return nil
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
	err := t.txn.SelectOne(l, "SELECT * FROM lobbys WHERE title = $1", l.Title)
	if err != nil {
		fmt.Println(err)
		t.txn.Rollback()
		return false
	}
	return true
}

func (t *AppTest) userShouldBeCreated(u *models.User) bool {
	err := t.txn.SelectOne(u, "SELECT * FROM users WHERE username = $1 AND email = $2", u.Username, u.Email)
	if err != nil {
		fmt.Println(err)
		t.txn.Rollback()
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

func (t *AppTest) TestStartAndEndAndRateLobby() {
	u, pp, err := insertTestUser(t.txn)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	resetTxn(t.txn, err)
	d := url.Values{}
	d.Add("username", u.Username)
	d.Add("password", pp)
	fmt.Println("user should be created")
	t.Assert(t.userShouldBeCreated(u))
	t.PostForm("/login", d)
	t.AssertOk()
	fmt.Println("inserting games")
	err = insertTestGames(t.txn)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	resetTxn(t.txn, err)
	fmt.Println("inserting lobby")
	l, err := insertTestLobby(t.txn, u)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	resetTxn(t.txn, err)
	t.Assert(t.lobbyShouldBeCreated(l))
	fmt.Println(l)
	t.Get(fmt.Sprintf("/lobby/switch/%d/start", l.Id)) // TODO: Use fmt.Sprintf
	t.AssertOk()
	t.AssertStatus(200)
	t.Get(fmt.Sprintf("/lobby/switch/%d/end", l.Id)) // TODO: Use fmt.Sprintf
	t.AssertOk()
}

func (t *AppTest) TestCreateLobby() {
	u := createTestUser()
	pp := u.Password // save plain-text password for later use --> http post
	u.HashPassword() // hash password before insert
	err := t.txn.Insert(u)
	if err != nil {
		t.txn.Rollback()
		panic(err)
		fmt.Println("panic B")
	}
	err = t.txn.Commit()
	if err != nil {
		fmt.Println("panic K")
		panic(err)
	}
	t.txn, err = Dbm.Begin()
	if err != nil {
		fmt.Println(err)
		panic(err)
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
	t.txn.Commit()
	t.txn, err = Dbm.Begin()
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	d.Add("username", u.Username)
	d.Add("password", pp)
	fmt.Println("user should be created")
	t.Assert(t.userShouldBeCreated(u))
	t.PostForm("/login", d)
	t.AssertOk()
	if err != nil {
		fmt.Println(err)
		t.Assert(false)
	}
	l := createTestLobby()
	d = url.Values{}
	d.Add("lobby.GameId", "3")
	d.Add("lobby.Title", l.Title)
	d.Add("lobby.Access", "public")
	d.Add("lobby.SkillLevel", "128")
	d.Add("lobby.MaxUsers", "32")
	d.Add("lobby.EstimatedPlayTime", "45")
	d.Add("lobby.EstimatedStartTime", "18:32")
	t.PostForm("/lobby/create", d)
	t.Assert(t.lobbyShouldBeCreated(l))
	fmt.Println(l)
	t.AssertOk()
	t.AssertStatus(200)
}

func (t *AppTest) After() {
	println("Tear down")
	t.txn.Commit()
	tearDown()
}
