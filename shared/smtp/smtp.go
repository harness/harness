package smtp

type SMTP struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	From     string `json:"from"`
	Username string `json:"username"`
	Password string `json:"password"`
}
