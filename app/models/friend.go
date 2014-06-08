package models

import (
	"time"
)

type Friend struct {
	Id     int64
	User   int64
	Friend int64
	Since  time.Time
}
