package model

type Token struct {
	AccessToken  string
	RefreshToken string
	Expiry       int64
}
