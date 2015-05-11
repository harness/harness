package common

type Token struct {
	ID     int64  `meddler:"token_id,pk"   json:"-"`
	UserID int64  `meddler:"user_id"       json:"-"`
	Login  string `meddler:"-"             json:"-"`
	Kind   string `meddler:"token_kind"    json:"kind,omitempty"`
	Label  string `meddler:"token_label"   json:"label,omitempty"`
	Expiry int64  `meddler:"token_expiry"  json:"expiry,omitempty"`
	Issued int64  `meddler:"token_issued"  json:"issued_at,omitempty"`
}

const (
	TokenUser  = "u"
	TokenSess  = "s"
	TokenHook  = "h"
	TokenAgent = "a"
)
