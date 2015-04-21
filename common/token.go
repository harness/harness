package common

const (
	TokenUser = "u"
	TokenSess = "s"
)

type Token struct {
	Kind   string   `json:"kind"`
	Login  string   `json:"-"`
	Label  string   `json:"label"`
	Repos  []string `json:"repos,omitempty"`
	Scopes []string `json:"scopes,omitempty"`
	Expiry int64    `json:"expiry,omitempty"`
	Issued int64    `json:"issued_at,omitempty"`
}
