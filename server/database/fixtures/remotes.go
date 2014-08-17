package fixtures

import (
	"github.com/drone/drone/shared/model"
	"github.com/jinzhu/gorm"
)

func LoadRemotes(db *gorm.DB) {
	db.Table("remotes").Create(model.Remote{
		Type:   "enterprise.github.com",
		Host:   "github.drone.io",
		Url:    "https://github.drone.io",
		Api:    "https://github.drone.io/v3/api",
		Client: "f0b461ca586c27872b43a0685cbc2847",
		Secret: "976f22a5eef7caacb7e678d6c52f49b1",
		Open:   true,
	})

	db.Table("remotes").Create(model.Remote{
		Type:   "github.com",
		Host:   "github.com",
		Url:    "https://github.io",
		Api:    "https://api.github.com",
		Client: "a0b461ca586c27872b43a0685cbc2847",
		Secret: "a76f22a5eef7caacb7e678d6c52f49b1",
		Open:   false,
	})
}
