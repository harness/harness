package session

type Session struct {
	URI         string `json:"uri"`
	ExpiresAt   int64  `json:"expiresAt"`
	AccessToken string `json:"accessToken"`
}
