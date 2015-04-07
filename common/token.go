package common

type Token struct {
	Sha    string   `json:"-"`
	Login  string   `json:"-"`
	Repos  []string `json:"repos,omitempty"`
	Scopes []string `json:"scopes,omitempty"`
	Expiry int64    `json:"expiry,omitempty"`
}
