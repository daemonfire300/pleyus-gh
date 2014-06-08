package models

type Game struct {
	Id    int64
	Name  string
	Genre string
}

func NewGame(name string, genre string) *Game {
	return &Game{
		Name:  name,
		Genre: genre,
	}
}
