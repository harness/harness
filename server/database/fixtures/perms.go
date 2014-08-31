package fixtures

import (
	"github.com/drone/drone/shared/model"
	"github.com/jinzhu/gorm"
)

func LoadPerms(db *gorm.DB) {
	db.Table("perms").Create(model.Perm{
		UserId:  101,
		RepoId:  200,
		Read:    true,
		Write:   true,
		Admin:   true,
		Created: 1398065343,
		Updated: 1398065344,
	})

	db.Table("perms").Create(model.Perm{
		UserId:  102,
		RepoId:  200,
		Read:    true,
		Write:   true,
		Admin:   false,
		Created: 1398065343,
		Updated: 1398065344,
	})

	db.Table("perms").Create(model.Perm{
		UserId:  103,
		RepoId:  200,
		Read:    true,
		Write:   false,
		Admin:   false,
		Created: 1398065343,
		Updated: 1398065344,
	})

	db.Table("perms").Create(model.Perm{
		UserId:  1,
		RepoId:  1,
		Read:    true,
		Write:   true,
		Admin:   true,
		Created: 1398065343,
		Updated: 1398065344,
	})

	db.Table("perms").Create(model.Perm{
		UserId:  1,
		RepoId:  2,
		Read:    true,
		Write:   true,
		Admin:   false,
		Created: 1398065343,
		Updated: 1398065344,
	})
}
