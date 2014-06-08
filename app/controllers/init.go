package controllers

import (
	"github.com/revel/revel"
)

func init() {
	revel.OnAppStart(InitDB)
	revel.InterceptMethod((*DatabaseController).Begin, revel.BEFORE)
	revel.InterceptMethod((*DatabaseController).Commit, revel.AFTER)
	revel.InterceptMethod((*DatabaseController).Rollback, revel.FINALLY)
	revel.InterceptMethod((*UserController).CheckUser, revel.BEFORE)
}
