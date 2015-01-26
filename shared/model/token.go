package model

import (
	"time"
)

type Token struct {
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
}
