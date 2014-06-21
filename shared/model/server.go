package model

type Server struct {
	ID   int64  `meddler:"server_id,pk" json:"id"`
	Name string `meddler:"server_name"  json:"name"`
	Host string `meddler:"server_host"  json:"host"`
	User string `meddler:"server_user"  json:"user"`
	Pass string `meddler:"server_pass"  json:"name"`
	Cert string `meddler:"server_cert"  json:"cert"`
}

type SMTPServer struct {
	ID   int64  `meddler:"smtp_id,pk" json:"id"`
	From string `meddler:"smtp_from"  json:"from"`
	Host string `meddler:"smtp_host"  json:"host"`
	Port string `meddler:"smtp_host"  json:"port"`
	User string `meddler:"smtp_user"  json:"user"`
	Pass string `meddler:"smtp_pass"  json:"name"`
}
