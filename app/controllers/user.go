package controllers

import (
	"database/sql"
	"errors"
	"strconv"

	"bitbucket.org/daemonfire300/pleyus-alpha/app/models"
	"github.com/revel/revel"
)

type UserController struct {
	DatabaseController
}

func (c UserController) getUser() (*models.User, error) {
	userIdStr, ok := c.Session["user"]
	if !ok {
		return nil, errors.New("No valid session given")
	}
	id, err := strconv.ParseInt(userIdStr, 10, 64)
	if err != nil {
		revel.INFO.Println(err)
		return nil, err
	}
	user, err := c.GetUserById(id)
	if err != nil {
		revel.INFO.Println(err)
		return nil, err
	}
	return user, nil
}

func (c UserController) authUser() revel.Result {
	_, err := c.getUser()
	if err != nil {
		return c.Redirect("/login")
	}
	return nil
}

func (c UserController) CheckUser() revel.Result {
	user, err := c.getUser()
	if err != nil {
		c.RenderArgs["isAuth"] = false
	} else {
		c.RenderArgs["isAuth"] = true
		c.RenderArgs["authUser"] = user
	}
	/*revel.INFO.Println("isAuth: ", c.RenderArgs["isAuth"])
	revel.INFO.Println("user: ", user)
	revel.INFO.Println("err: ", err)*/
	return nil
}

func (c UserController) Logout() revel.Result {
	for k := range c.Session {
		delete(c.Session, k)
	}
	return c.Redirect(App.Index)
}

func (c UserController) DoRegister(user models.User) revel.Result {
	user.Validate(c.Validation)
	if c.Validation.HasErrors() {
		revel.INFO.Println("errors detected")
		revel.INFO.Println(c.Validation.Errors)
		c.Validation.Keep()
		c.FlashParams()
		return c.Redirect("/register")
	}
	user.HashPassword()
	c.SaveUser(&user)
	return c.Redirect("/register")
}

func (c UserController) Register() revel.Result {
	return c.Render()
}

func (c UserController) Login(username string, password string) revel.Result {
	return c.Render()
}

func (c UserController) DoLogin(username string, password string) revel.Result {
	user, err := c.GetUserByName(username)
	if err != nil {
		// user does not exist (or other error)
		revel.INFO.Println(err)
		return c.Redirect("/login?notfound")
	}
	user.ValidatePassword(c.Validation, password)
	if c.Validation.HasErrors() {
		revel.INFO.Println("errors detected")
		revel.INFO.Println(c.Validation.Errors)
		c.Validation.Keep()
		c.FlashParams()
		return c.Redirect("/login")
	} else {
		c.Session["user"] = strconv.FormatInt(user.Id, 10)
		c.Session.SetDefaultExpiration()
	}
	return c.Redirect("/user/profile")
}

func (c UserController) ViewProfile(userid int64) revel.Result {
	user, err := c.GetUserById(userid)
	if err != nil && err != sql.ErrNoRows {
		revel.ERROR.Println(err)
		panic(err)
	}
	if err == sql.ErrNoRows {
		c.Flash.Error("User not found")
		return c.Redirect(App.Index)
	}
	return c.Render(user)
}

func (c UserController) Profile() revel.Result {
	user, _ := c.getUser()
	//user, _ := c.GetUserByName("daemonfire")
	return c.Render(user)
}
