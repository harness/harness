package model

type Netrc struct {
	Machine  string `json:"machine"`
	Login    string `json:"login"`
	Password string `json:"password"`
}
