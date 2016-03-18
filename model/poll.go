package model

//Poll git poll
type Poll struct {
	ID     int64  `json:"id"                meddler:"id,pk"`
	Owner  string `json:"owner"             meddler:"owner"`
	Name   string `json:"name"              meddler:"name"`
	Period uint64 `json:"period"            meddler:"period"`
}
