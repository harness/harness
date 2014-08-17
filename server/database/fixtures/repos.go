package fixtures

import (
	"github.com/drone/drone/shared/model"
	"github.com/jinzhu/gorm"
)

func LoadRepos(db *gorm.DB) {
	db.Table("repos").Create(model.Repo{
		UserId:      0,
		Remote:      "github.com",
		Host:        "github.com",
		Owner:       "lhofstadter",
		Name:        "lenwoloppali",
		CloneUrl:    "git://github.com/lhofstadter/lenwoloppali.git",
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

	db.Table("repos").Create(model.Repo{
		UserId:      0,
		Remote:      "github.com",
		Host:        "github.com",
		Owner:       "browndynamite",
		Name:        "lenwoloppali",
		CloneUrl:    "git://github.com/browndynamite/lenwoloppali.git",
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

	db.Table("repos").Create(model.Repo{
		UserId:      0,
		Remote:      "gitlab.com",
		Host:        "gitlab.com",
		Owner:       "browndynamite",
		Name:        "lenwoloppali",
		CloneUrl:    "git://gitlab.com/browndynamite/lenwoloppali.git",
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
