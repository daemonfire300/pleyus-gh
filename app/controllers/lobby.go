package controllers

import (
	"database/sql"
	"strings"
	"time"

	"bitbucket.org/daemonfire300/pleyus-alpha/app/models"
	"github.com/revel/revel"
)

type LobbyController struct {
	UserController
}

func (c LobbyController) Index() revel.Result {
	return c.Render()
}

func (c LobbyController) List(game string, title string) revel.Result {
	var lobbys []*models.Lobby
	var queryParts []string
	searchQuery := "  "
	var gameid int64

	if game != "" {
		var g models.Game
		err := c.Txn.SelectOne(&g, "SELECT id FROM games WHERE name = $1", game)
		if err != nil {
			revel.INFO.Println(err)
		} else {
			gameid = g.Id
		}
	}
	if gameid > 0 {
		queryParts = append(queryParts, " gameid = :gameid ")
	}
	if title != "" {
		queryParts = append(queryParts, " title % :title ") // % needs pg_trgm
		// CREATE EXTENSION pg_trgm; --> http://www.rdegges.com/easy-fuzzy-text-searching-with-postgresql/
	}
	if len(queryParts) > 0 {
		searchQuery += " AND "
		searchQuery += strings.Join(queryParts, " AND ")
		revel.INFO.Println("Using search parameters", game, title, " generated query ", searchQuery)
	}
	results, err := c.Txn.Select(models.Lobby{}, "SELECT * FROM lobbys WHERE access <> :access AND state = :state "+searchQuery, map[string]interface{}{
		"gameid": gameid,
		"title":  title,
		"access": "private",
		"state":  "open",
	})
	revel.INFO.Println("gameid", gameid)
	revel.INFO.Println("title", title)
	revel.INFO.Println(searchQuery)
	if err != nil {
		revel.INFO.Println(err)
	} else {
		for _, lobby := range results {
			lobbys = append(lobbys, lobby.(*models.Lobby))
		}
	}
	return c.Render(lobbys)
}

func (c LobbyController) Join(lobbyid int64) revel.Result {
	user, err := c.getUser()
	if err != nil {
		return c.Redirect(UserController.Login)
	}
	if user.HasLobby(c.Txn) {
		return c.RenderJson("you already are part of a lobby")
	}

	if err != nil {
		revel.ERROR.Println(err)
		return c.RenderJson(err.Error())
	}
	var lobby *models.Lobby
	lobby, err = c.GetLobbyById(lobbyid)
	if err != nil {
		revel.ERROR.Println(err)
		return c.RenderJson(err.Error())
	}
	lobby.GetPlayers(c.Txn)
	if lobby.IsFull() {
		c.Flash.Error("Lobby already full")
		return c.Redirect(App.Index)
	}
	if lobby.State == "closed" {
		c.Flash.Error("Lobby is closed")
		return c.Redirect(App.Index)
	}
	if lobby.State == "starting" {
		c.Flash.Error("Lobby lobby has already been started")
		return c.Redirect(App.Index)
	}
	user.Lobby = lobby
	c.UpdateUser(user)
	err = c.AddUserToLobby(user, lobby)
	if err != nil {
		c.Flash.Error("Something went wrong (adding user to lobby)")
		return c.Redirect(App.Index)
	}
	return c.Redirect("/lobby/view/%d", lobbyid)
}

func (c LobbyController) isValidState(state string) bool {
	validStates := [...]string{"closed", "open"}
	for _, st := range validStates {
		if state == st {
			return true
		}
	}
	return false
}

func (c LobbyController) isLobbyOwnerFlash(user *models.User, lobbyid int64) bool {
	// TODO: This does not actually indicate if you are the lobbyowner.... --> FIXED
	// get lobby
	if !user.HasLobby(c.Txn) {
		revel.INFO.Println("user ", user, " has no lobby")
		c.Flash.Error("You are not in a lobby")
		return false
	}
	if user.Lobby.OwnerId != user.Id {
		c.Flash.Error("You are not the lobby owner")
		return false
	}
	return true
}

func (c LobbyController) State(lobbyid int64, state string) revel.Result {
	user, err := c.getUser()
	if err != nil {
		return c.Redirect(UserController.Login)
	}
	if !c.isLobbyOwnerFlash(user, lobbyid) {
		return c.Redirect("/lobby/view/%d", lobbyid)
	}
	var lobby *models.Lobby
	lobby, err = c.GetLobbyById(lobbyid)
	if err != nil && err != sql.ErrNoRows {
		revel.ERROR.Println(err)
		panic(err)
	}
	if err == sql.ErrNoRows {
		c.Flash.Error("Lobby not found")
		return c.Redirect(App.Index)
	}
	if lobby.State == "closed" {
		c.Flash.Success("Lobby already closed")
		return c.Redirect("/lobby/view/%d", lobbyid)
	}
	if lobby.State == "done" || lobby.State == "started" {
		c.Flash.Success("Lobby already done or started")
		return c.Redirect("/lobby/view/%d", lobbyid)
	}
	c.Validation.Required(state)
	lobby.State = state
	c.UpdateLobby(lobby)
	return c.Redirect("/lobby/view/%d", lobbyid)
}

func (c LobbyController) startLobby(lobby *models.Lobby) {
	//time.Sleep(time.Millisecond * 5000 * 0)
	lobby.State = "started"
	//c.DatabaseController.Begin()
	c.UpdateLobby(lobby)
	//c.DatabaseController.Commit()
	//c.DatabaseController.Rollback()
}

func (c LobbyController) StartOrEndLobby(lobbyid int64, a string) revel.Result {
	user, err := c.getUser()
	if err != nil {
		return c.Redirect(UserController.Login)
	}
	if !c.isLobbyOwnerFlash(user, lobbyid) {
		return c.Redirect("/lobby/view/%d", lobbyid)
	}
	var lobby *models.Lobby
	lobby, err = c.GetLobbyById(lobbyid)
	if err != nil && err != sql.ErrNoRows {
		revel.INFO.Println(err)
	}
	if err == sql.ErrNoRows {
		c.Flash.Error("Lobby not found")
		return c.Redirect(App.Index)
	}

	if a == "start" {
		if lobby.State == "done" || lobby.State == "started" {
			c.Flash.Success("Lobby already done or started")
			return c.Redirect("/lobby/view/%d", lobbyid)
		}
		lobby.State = "starting"
	} else {
		if lobby.State != "started" {
			c.Flash.Success("Lobby already done or has not been started")
			return c.Redirect("/lobby/rate/%d", lobbyid)
		}
		/*if lobby.State == "done"{
			c.Flash.Success("Lobby already done or started")
			return c.Redirect("/lobby/view/%d", lobbyid)
		}*/

		lobby.State = "done"
	}
	c.UpdateLobby(lobby)
	if a == "start" {
		//go c.startLobby(lobby)
		c.startLobby(lobby)
	}
	return c.Redirect("/lobby/view/%d", lobbyid)
}

// rs : ratings
func (c LobbyController) Rate(lobbyid int64) revel.Result {
	var rs map[int64]int
	c.Params.Bind(&rs, "rs")
	user, err := c.getUser()
	if err != nil {
		return c.Redirect(UserController.Login)
	}
	var lobby *models.Lobby
	lobby, err = c.GetLobbyById(lobbyid)
	if err != nil && err != sql.ErrNoRows {
		revel.ERROR.Println(err)
		panic(err)
	}
	if err == sql.ErrNoRows {
		c.Flash.Error("Lobby not found")
		revel.INFO.Println("Lobby not found")
		return c.Redirect(App.Index)
	}
	// only rate lobby which has been started --> then --> ended
	if !lobby.Ended() {
		c.Flash.Error("Lobby still ongoing")
		revel.INFO.Println("Lobby still ongoing")
		return c.Redirect("/lobby/view/%d", lobbyid)
	}
	// check if user is actually participating in the lobby
	if !lobby.HasPlayer(c.Txn, user) {
		c.Flash.Error("Not part of this lobby")
		revel.INFO.Println("User not part of this lobby")
		return c.Redirect(App.Index)
	}
	lobby.GetPlayers(c.Txn)
	ps := lobby.Players
	// method == GET	: Display other participants and general lobby rating functionality
	if c.Request.Method == "GET" {
		return c.Render(lobbyid, ps)
	}
	// method == POST	: Get ratings and apply them
	lr, ok := rs[0]
	if ok && lobby.IsValidRating(lr) {
		lobby.Rating += lr
	}
	c.Txn.Update(lobby)
	for k, v := range rs {
		p, ok := ps[k]
		if !ok || k == user.Id {
			continue
		}
		p.ApplyRating(v)
		aff, err := c.Txn.Update(p)
		revel.INFO.Println("Affected rows (raw): ", aff)
		if err != nil {
			revel.INFO.Println(err)
			return c.Redirect("/lobby/rate/%d", lobbyid)
		}
	}
	err = user.FinishLobby(c.Txn)
	if err != nil {
		revel.INFO.Println(err)
	}
	// user send ratings, time to clink him out of the lobby and mark it as rated and inactive

	return c.Redirect(App.Index)
}

func (c LobbyController) EditMeta(lobbyid int64, m models.LobbyMeta) revel.Result {
	user, err := c.getUser()
	if err != nil {
		return c.Redirect(UserController.Login)
	}
	if c.isLobbyOwnerFlash(user, lobbyid) {
		return c.Redirect("/lobby/view/%d", lobbyid)
	}
	l, err := c.GetLobbyById(lobbyid)
	if err != nil {
		revel.INFO.Println(err)
		c.Flash.Error("Lobby not found")
		return c.Redirect(App.Index)
	}
	// find any existing meta linked to the lobby
	if c.Request.Method == "POST" {
		c.SaveLobbyMeta(&m)
	} else {
		mt, err := l.GetMeta(c.Txn)
		if err != sql.ErrNoRows {
			m = models.LobbyMeta{}
		} else {
			m = *mt
		}
		if err != nil && err != sql.ErrNoRows {
			revel.INFO.Println(err)
			return c.Redirect(App.Index)
		}
	}

	return c.Render(lobbyid, m)
}

func (c LobbyController) ViewLobby(lobbyid int64) revel.Result {
	user, err := c.getUser()
	if err != nil {
		return c.Redirect(UserController.Login)
	}
	var lobby *models.Lobby

	lobby, err = c.GetLobbyById(lobbyid)
	//err = c.Txn.SelectOne(&lobby, "SELECT * FROM lobbys WHERE id = $1", lobbyid)
	if err != nil && err != sql.ErrNoRows {
		revel.ERROR.Println(err)
		panic(err)
	}
	if err == sql.ErrNoRows {
		c.Flash.Error("Lobby not found")
		return c.Redirect(App.Index)
	}
	canJoinLobby := false
	canViewLobby := true
	user.GetLobby(c.Txn)
	inLobby := user.InLobby(lobbyid)
	hasPassword := (lobby.Password.String != "")
	lobby.GetPlayers(c.Txn)
	if !user.HasLobby(c.Txn) {
		canJoinLobby = true
	}
	if canJoinLobby && lobby.Access == "private" && lobby.OwnerId != user.Id {
		canViewLobby = false
	}
	// get lobbymeta (server ip, description etc)
	var meta *models.LobbyMeta
	meta, err = lobby.GetMeta(c.Txn)
	startsInMinutes := int(lobby.EstimatedStartTime.Sub(time.Now()).Minutes())
	return c.Render(lobby, canJoinLobby, canViewLobby, hasPassword, startsInMinutes, meta, inLobby)
}

func (c LobbyController) KickPlayer(userid int64, lobbyid int64) revel.Result {
	user, err := c.getUser()
	if err != nil {
		return c.Redirect(UserController.Login)
	}
	if user.Id == userid {
		c.Flash.Error("You can not kick yourself")
		return c.Redirect(c.Request.Referer())
	}
	user.GetLobby(c.Txn)
	if !user.IsOwner() {
		c.Flash.Error("You are not the owner of the lobby, therefore you have no administrative priviliges")
		return c.Redirect(c.Request.Referer())
	}
	target, err := c.GetUserById(userid)
	if err != nil {
		revel.INFO.Println(err)
		c.Flash.Error("Error")
		return c.Redirect(c.Request.Referer())
	}
	target.RemoveCurrentLobby(c.Txn)
	c.Flash.Success("Kicked Player " + target.Username)
	return c.Redirect(c.Request.Referer())
}

func (c LobbyController) InvitePlayer(userid int64) revel.Result {
	user, err := c.getUser()
	if err != nil {
		return c.Redirect(UserController.Login)
	}
	if !user.HasLobby(c.Txn) {
		c.Flash.Error("You can not invite someone to a lobby you are not a part of")
		return c.Redirect(c.Request.Referer())
	}
	if user.Id == userid {
		c.Flash.Error("You can not invite yourself")
		return c.Redirect(c.Request.Referer())
	}
	targetUser, err := c.GetUserById(userid)
	if err != nil {
		c.Flash.Error("User not found")
		return c.Redirect(c.Request.Referer())
	}
	targetUser.GetLobby(c.Txn)
	if targetUser.InLobby(user.Lobby.Id) {
		c.Flash.Error("User is already in the lobby")
		return c.Redirect(c.Request.Referer())
	}

	invite := models.LobbyInvite{
		From:    user.Id,
		To:      userid,
		LobbyId: user.Lobby.Id,
		Status:  "unanswered",
	}
	err = c.Txn.SelectOne(&invite, "SELECT * FROM lobbyinvites WHERE from=$1 AND to=$2 AND lobbyid=$3 AND status=$4", invite.From, invite.To, invite.LobbyId, invite.Status)
	if err == sql.ErrNoRows {
		err = c.Txn.Insert(invite)
	}
	if err != nil {
		revel.INFO.Println(err)
		c.Flash.Error("Error occured while sending invite")
		return c.Redirect(c.Request.Referer())
	}
	c.Flash.Success("Invitation sent")
	return c.Redirect(c.Request.Referer())
}

func (c LobbyController) Leave(lobbyid int64) revel.Result {
	user, err := c.getUser()
	if err != nil {
		return c.Redirect(UserController.Login)
	}
	user.GetLobby(c.Txn)
	if !user.InLobby(lobbyid) {
		return c.Redirect(App.Index)
	}
	if user.IsOwner() {
		user.Lobby.GetPlayers(c.Txn)
		l := *(user.Lobby)
		for k, _ := range user.Lobby.Players {
			err = user.Lobby.Players[k].RemoveCurrentLobby(c.Txn)
		}
		_, err = c.Txn.Delete(l)
	} else {
		err = user.RemoveCurrentLobby(c.Txn)
	}
	if err != nil {
		revel.INFO.Println(err)
	}
	c.Flash.Success("Left the lobby")
	return c.Redirect(App.Index)
}

func (c LobbyController) Create() revel.Result {
	games, err := c.Txn.Select(models.Game{}, "SELECT * FROM games ORDER BY name ASC")
	if err != nil && err != sql.ErrNoRows {
		revel.ERROR.Println(err)
		panic(err)
	}
	return c.Render(games)
}

func (c LobbyController) GetGames() revel.Result {
	games, err := c.Txn.Select(models.Game{}, "SELECT * FROM games ORDER BY name ASC")
	if err != nil && err != sql.ErrNoRows {
		revel.INFO.Println(err)
	}
	return c.RenderJson(games)
}

func (c LobbyController) PostCreate(lobby models.Lobby) revel.Result {
	if auth := c.RenderArgs["isAuth"]; auth == false {
		return c.Redirect(UserController.Login)
	}
	user := c.RenderArgs["authUser"].(*models.User)
	if user.HasLobby(c.Txn) {
		c.Validation.Error("You are already part of a lobby")
	}
	revel.INFO.Println(c.Request.FormValue("lobby.EstimatedStartTime"))
	lobby.Validate(c.Validation)
	layout := "15:04"
	ets, err := time.Parse(layout, c.Request.FormValue("lobby.EstimatedStartTime"))
	if err != nil {
		revel.INFO.Println(err)
		c.Validation.Error("Not a valid date").Key("lobby.EstimatedStartTime")
	} else {
		now := time.Now()
		ets = time.Date(now.Year(), now.Month(), now.Day(), ets.Hour(), ets.Minute(), 0, 0, now.Location())
		lobby.EstimatedStartTime = ets
	}
	revel.INFO.Println("EstimatedStartTime", lobby.EstimatedStartTime)
	if c.Validation.HasErrors() {
		revel.INFO.Println("errors detected")
		revel.INFO.Println(c.Validation.Errors)
		c.Validation.Keep()
		c.FlashParams()
		return c.Redirect(LobbyController.Create)
	}

	lobby.State = "open"
	lobby.Owner = user
	c.SaveLobby(&lobby)
	user.Lobby = &lobby
	c.SaveUser(user)
	err = c.AddUserToLobby(user, &lobby)
	if err != nil {
		c.Flash.Error("Something went wrong (adding user to lobby)")
		return c.Redirect(App.Index)
	}

	return c.Redirect(App.Index)
}
