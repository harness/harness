package fixtures

import (
	"database/sql"

	"github.com/drone/drone/shared/model"
	"github.com/russross/meddler"
)

func LoadRepos(db *sql.DB) {
	meddler.Insert(db, "repos", &model.Repo{
		UserID:      0,
		Remote:      "github.com",
		Host:        "github.com",
		Owner:       "lhofstadter",
		Name:        "lenwoloppali",
		CloneURL:    "git://github.com/lhofstadter/lenwoloppali.git",
		Active:      true,
		Private:     true,
		Privileged:  true,
		PostCommit:  true,
		PullRequest: true,
		PublicKey:   "publickey",
		PrivateKey:  "privatekey",
		Params:      "params",
		Timeout:     900,
		Created:     1398065343,
		Updated:     1398065344,
	})

	meddler.Insert(db, "repos", &model.Repo{
		UserID:      0,
		Remote:      "github.com",
		Host:        "github.com",
		Owner:       "browndynamite",
		Name:        "lenwoloppali",
		CloneURL:    "git://github.com/browndynamite/lenwoloppali.git",
		Active:      true,
		Private:     true,
		Privileged:  true,
		PostCommit:  true,
		PullRequest: true,
		PublicKey:   "publickey",
		PrivateKey:  "privatekey",
		Params:      "params",
		Timeout:     900,
		Created:     1398065343,
		Updated:     1398065344,
	})

	meddler.Insert(db, "repos", &model.Repo{
		UserID:      0,
		Remote:      "gitlab.com",
		Host:        "gitlab.com",
		Owner:       "browndynamite",
		Name:        "lenwoloppali",
		CloneURL:    "git://gitlab.com/browndynamite/lenwoloppali.git",
		Active:      true,
		Private:     true,
		Privileged:  true,
		PostCommit:  true,
		PullRequest: true,
		PublicKey:   "publickey",
		PrivateKey:  "privatekey",
		Params:      "params",
		Timeout:     900,
		Created:     1398065343,
		Updated:     1398065344,
	})
}
