package fixtures

import (
	"database/sql"

	"github.com/drone/drone/shared/model"
	"github.com/russross/meddler"
)

func LoadPerms(db *sql.DB) {
	meddler.Insert(db, "perms", &model.Perm{
		UserID:  101,
		RepoID:  200,
		Read:    true,
		Write:   true,
		Admin:   true,
		Created: 1398065343,
		Updated: 1398065344,
	})

	meddler.Insert(db, "perms", &model.Perm{
		UserID:  102,
		RepoID:  200,
		Read:    true,
		Write:   true,
		Admin:   false,
		Created: 1398065343,
		Updated: 1398065344,
	})

	meddler.Insert(db, "perms", &model.Perm{
		UserID:  103,
		RepoID:  200,
		Read:    true,
		Write:   false,
		Admin:   false,
		Created: 1398065343,
		Updated: 1398065344,
	})

	meddler.Insert(db, "perms", &model.Perm{
		UserID:  1,
		RepoID:  1,
		Read:    true,
		Write:   true,
		Admin:   true,
		Created: 1398065343,
		Updated: 1398065344,
	})

	meddler.Insert(db, "perms", &model.Perm{
		UserID:  1,
		RepoID:  2,
		Read:    true,
		Write:   true,
		Admin:   false,
		Created: 1398065343,
		Updated: 1398065344,
	})
}
