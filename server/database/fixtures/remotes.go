package fixtures

import (
	"database/sql"

	"github.com/drone/drone/shared/model"
	"github.com/russross/meddler"
)

func LoadRemotes(db *sql.DB) {
	meddler.Insert(db, "remotes", &model.Remote{
		Type:   "enterprise.github.com",
		Host:   "github.drone.io",
		URL:    "https://github.drone.io",
		API:    "https://github.drone.io/v3/api",
		Client: "f0b461ca586c27872b43a0685cbc2847",
		Secret: "976f22a5eef7caacb7e678d6c52f49b1",
		Open:   true,
	})

	meddler.Insert(db, "remotes", &model.Remote{
		Type:   "github.com",
		Host:   "github.com",
		URL:    "https://github.io",
		API:    "https://api.github.com",
		Client: "a0b461ca586c27872b43a0685cbc2847",
		Secret: "a76f22a5eef7caacb7e678d6c52f49b1",
		Open:   false,
	})
}
