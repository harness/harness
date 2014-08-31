package model

type Server struct {
	Id   int64  `gorm:"primary_key:yes" json:"id"`
	Name string `json:"name"`
	Host string `json:"host"`
	User string `json:"user"`
	Pass string `json:"name"`
	Cert string `json:"cert"`
}

type SMTPServer struct {
	Id   int64  `gorm:"primary_key:yes" json:"id"`
	From string `json:"from"`
	Host string `json:"host"`
	Port string `json:"port"`
	User string `json:"user"`
	Pass string `json:"name"`
}
