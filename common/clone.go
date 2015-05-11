package common

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

// Keypair represents an RSA public and private key
// assigned to a repository. It may be used to clone
// private repositories, or as a deployment key.
type Keypair struct {
	Public  string `json:"public,omitempty"`
	Private string `json:"private,omitempty"`
}
