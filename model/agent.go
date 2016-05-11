package model

type Agent struct {
	ID       int64  `json:"id"         meddler:"agent_id,pk"`
	Address  string `json:"address"    meddler:"agent_addr"`
	Platform string `json:"platform"   meddler:"agent_platform"`
	Capacity int    `json:"capacity"   meddler:"agent_capacity"`
	Created  int64  `json:"created_at" meddler:"agent_created"`
	Updated  int64  `json:"updated_at" meddler:"agent_updated"`
}
