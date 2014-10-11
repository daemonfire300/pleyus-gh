package controllers

import "github.com/revel/revel"

type App struct {
	UserController
}

func (c App) Index() revel.Result {
	//revel.WARN.Printf("%#v", revel.CodePaths)
	return c.Render()
}
