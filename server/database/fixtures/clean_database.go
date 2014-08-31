package fixtures

import (
	"github.com/jinzhu/gorm"
)

func CleanDatabase(db *gorm.DB) {
	db.Exec("DROP TABLE IF EXISTS commits;")
	db.Exec("DROP TABLE IF EXISTS perms;")
	db.Exec("DROP TABLE IF EXISTS users;")
	db.Exec("DROP TABLE IF EXISTS repos;")
	db.Exec("DROP TABLE IF EXISTS output;")
	db.Exec("DROP TABLE IF EXISTS remotes;")
	db.Exec("DROP TABLE IF EXISTS servers;")
	db.Exec("DROP TABLE IF EXISTS smtp;")

	db.Exec("DROP TABLE IF EXISTS migration;")
}
