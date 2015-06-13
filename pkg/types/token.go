package types

type Token struct {
	ID     int64  `meddler:"token_id,pk"   json:"-"`
	UserID int64  `meddler:"token_user_id" json:"-"                sql:"index:ix_token_user_id,unique:ux_token_user_label"`
	Login  string `meddler:"-"             json:"-"                sql:"-"`
	Kind   string `meddler:"token_kind"    json:"kind,omitempty"`
	Label  string `meddler:"token_label"   json:"label,omitempty"  sql:"unique:ux_token_user_label"`
	Expiry int64  `meddler:"token_expiry"  json:"expiry,omitempty"`
	Issued int64  `meddler:"token_issued"  json:"issued_at,omitempty"`
}

const (
	TokenUser  = "u"
	TokenSess  = "s"
	TokenHook  = "h"
	TokenAgent = "a"
)
