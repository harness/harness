package types

type Clone struct {
	Origin  string   `json:"origin"`
	Remote  string   `json:"remote"`
	Branch  string   `json:"branch"`
	Sha     string   `json:"sha"`
	Ref     string   `json:"ref"`
	Dir     string   `json:"dir"`
	Netrc   *Netrc   `json:"netrc"`
	Keypair *Keypair `json:"keypair"`
}

type Netrc struct {
	Machine  string `json:"machine"`
	Login    string `json:"login"`
	Password string `json:"user"`
}
