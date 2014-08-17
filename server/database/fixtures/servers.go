package fixtures

import (
	"github.com/drone/drone/shared/model"
	"github.com/jinzhu/gorm"
)

func LoadServers(db *gorm.DB) {
	db.Table("servers").Create(model.Server{
		Name: "docker1",
		Host: "tcp://127.0.0.1:4243",
		User: "root",
		Pass: "pa55word",
		Cert: "/path/to/cert.key",
	})

	db.Table("servers").Create(model.Server{
		Name: "docker2",
		Host: "tcp://127.0.0.1:4243",
		User: "root",
		Pass: "pa55word",
		Cert: "/path/to/cert.key",
	})
}
