package server

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/drone/drone/shared/envconfig"
)

type Server struct {
	Addr string
	Cert string
	Key  string
}

func Load(env envconfig.Env) *Server {
	return &Server{
		Addr: env.String("SERVER_ADDR", ":8000"),
		Cert: env.String("SERVER_CERT", ""),
		Key:  env.String("SERVER_KEY", ""),
	}
}

func (s *Server) Run(handler http.Handler) {
	log.Infof("starting server %s", s.Addr)

	if len(s.Cert) != 0 {
		log.Fatal(
			http.ListenAndServeTLS(s.Addr, s.Cert, s.Key, handler),
		)
	} else {
		log.Fatal(
			http.ListenAndServe(s.Addr, handler),
		)
	}
}
