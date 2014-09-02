package fixtures

import (
	"database/sql"

	"github.com/drone/drone/shared/model"
	"github.com/russross/meddler"
)

func LoadServers(db *sql.DB) {
	meddler.Insert(db, "servers", &model.Server{
		Name: "docker1",
		Host: "tcp://127.0.0.1:4243",
		User: "root",
		Pass: "pa55word",
		Cert: "/path/to/cert.key",
	})

	meddler.Insert(db, "servers", &model.Server{
		Name: "docker2",
		Host: "tcp://127.0.0.1:4243",
		User: "root",
		Pass: "pa55word",
		Cert: "/path/to/cert.key",
	})
}
