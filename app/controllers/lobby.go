package controllers

import (
	"database/sql"
	"github.com/daemonfire300/pleyusweb/app/models"
	"github.com/revel/revel"
	"time"
)

type LobbyController struct {
	UserController
}

func (c LobbyController) Index() revel.Result {
	return c.Render()
}

func (c LobbyController) List() revel.Result {
	var lobbys []*models.Lobby
	var searchQueryParts []string
	searchQuery = " WHERE "
	searchGame = revel.Request.FormValue("game")
	searchTitle = revel.Request.FormValue("title")

	if searchGame != "" {
		searchQueryParts = append(searchQueryParts, " gameid = :gameid ")
	}
	if searchTitle != "" {
		searchQueryParts = append(searchQueryParts, " title % :gameid ")
	}
	results, err := c.Txn.Select(models.Lobby{}, "SELECT * FROM lobbys")
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
	if user.LobbyId > 0 {
		return c.RenderJson("you already are part of a lobby")
	}

	if err != nil {
		revel.ERROR.Println(err)
		return c.RenderJson(err.Error())
	}
	var lobby models.Lobby
	err = c.Txn.SelectOne(&lobby, "SELECT * FROM lobbys WHERE id = $1", lobbyid)
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
	user.Lobby = &lobby
	c.UpdateUser(user)

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

func (c LobbyController) State(lobbyid int64, state string) revel.Result {
	user, err := c.getUser()
	if err != nil {
		return c.Redirect(UserController.Login)
	}
	if user.LobbyId != lobbyid {
		c.Flash.Error("You are not the lobby owner")
		return c.Redirect("/lobby/view/%d", lobbyid)
	}
	var lobby models.Lobby
	err = c.Txn.SelectOne(&lobby, "SELECT * FROM lobbys WHERE id = $1", lobbyid)
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
	c.UpdateLobby(&lobby)
	return c.Redirect("/lobby/view/%d", lobbyid)
}

func (c LobbyController) startLobby(lobby *models.Lobby) {
	time.Sleep(time.Millisecond * 5000)
	lobby.State = "started"
	c.DatabaseController.Begin()
	c.UpdateLobby(lobby)
	c.DatabaseController.Commit()
	c.DatabaseController.Rollback()
}

func (c LobbyController) StartLobby(lobbyid int64) revel.Result {
	user, err := c.getUser()
	if err != nil {
		return c.Redirect(UserController.Login)
	}
	if user.LobbyId != lobbyid {
		c.Flash.Error("You are not the lobby owner")
		return c.Redirect("/lobby/view/%d", lobbyid)
	}
	var lobby models.Lobby
	err = c.Txn.SelectOne(&lobby, "SELECT * FROM lobbys WHERE id = $1", lobbyid)
	if err != nil && err != sql.ErrNoRows {
		revel.ERROR.Println(err)
		panic(err)
	}
	if err == sql.ErrNoRows {
		c.Flash.Error("Lobby not found")
		return c.Redirect(App.Index)
	}

	if lobby.State == "done" || lobby.State == "started" {
		c.Flash.Success("Lobby already done or started")
		return c.Redirect("/lobby/view/%d", lobbyid)
	}

	lobby.State = "starting"
	c.UpdateLobby(&lobby)
	go c.startLobby(&lobby)
	return c.Redirect("/lobby/view/%d", lobbyid)
}

func (c LobbyController) ViewLobby(lobbyid int64) revel.Result {
	user, err := c.getUser()
	if err != nil {
		return c.Redirect(UserController.Login)
	}
	var lobby models.Lobby
	canJoinLobby := false
	canViewLobby := true
	hasPassword := (lobby.Password.String != "")

	err = c.Txn.SelectOne(&lobby, "SELECT * FROM lobbys WHERE id = $1", lobbyid)
	if err != nil && err != sql.ErrNoRows {
		revel.ERROR.Println(err)
		panic(err)
	}
	if err == sql.ErrNoRows {
		c.Flash.Error("Lobby not found")
		return c.Redirect(App.Index)
	}
	lobby.GetPlayers(c.Txn)
	if user.LobbyId == 0 {
		canJoinLobby = true
	}
	if canJoinLobby && lobby.Access == "private" && lobby.OwnerId != user.Id {
		canViewLobby = false
	}
	return c.Render(lobby, canJoinLobby, canViewLobby, hasPassword)
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
	target.LobbyId = 0
	c.UpdateUser(target)
	c.Flash.Success("Kicked Player " + target.Username)
	return c.Redirect(c.Request.Referer())
}

func (c LobbyController) InvitePlayer(userid int64) revel.Result {
	user, err := c.getUser()
	if err != nil {
		return c.Redirect(UserController.Login)
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
	if targetUser.LobbyId == user.LobbyId {
		c.Flash.Error("User is already in the lobby")
		return c.Redirect(c.Request.Referer())
	}

	invite := models.LobbyInvite{
		From:    user.Id,
		To:      userid,
		LobbyId: user.LobbyId,
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
	if user.LobbyId == lobbyid {
		if user.Id == user.Lobby.OwnerId {
			user.Lobby.GetPlayers(c.Txn)
			for k, plr := range user.Lobby.Players {
				user.Lobby.Players[k].LobbyId = 0
				user.Lobby.Players[k].Lobby = nil
				_, err = c.Txn.Update(plr)
			}
			_, err = c.Txn.Delete(user.Lobby)
		} else {
			user.LobbyId = 0
			user.Lobby = nil
			_, err = c.Txn.Update(user)
		}
		if err != nil {
			revel.INFO.Println(err)
		}
		c.Flash.Success("Left the lobby")
		return c.Redirect(App.Index)
	} else {
		return c.Redirect(App.Index)
	}
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
		revel.ERROR.Println(err)
		panic(err)
	}
	return c.RenderJson(games)
}

func (c LobbyController) DoCreate(lobby models.Lobby) revel.Result {
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
	}
	revel.INFO.Println("EstimatedStartTime", ets)
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

	return c.Redirect(App.Index)
}
