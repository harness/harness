package fixtures

import (
	"database/sql"

	"github.com/drone/drone/shared/model"
	"github.com/russross/meddler"
)

func LoadUsers(db *sql.DB) {
	meddler.Insert(db, "users", &model.User{
		Remote:   "github.com",
		Login:    "smellypooper",
		Access:   "f0b461ca586c27872b43a0685cbc2847",
		Secret:   "976f22a5eef7caacb7e678d6c52f49b1",
		Name:     "Dr. Cooper",
		Email:    "drcooper@caltech.edu",
		Gravatar: "b9015b0857e16ac4d94a0ffd9a0b79c8",
		Token:    "e42080dddf012c718e476da161d21ad5",
		Admin:    true,
		Active:   true,
		Syncing:  false,
		Created:  1398065343,
		Updated:  1398065344,
		Synced:   1398065345,
	})

	meddler.Insert(db, "users", &model.User{
		Remote:   "github.com",
		Login:    "lhofstadter",
		Access:   "e4105c3059ac4c466594932dc9a4ffb2",
		Secret:   "2257216903d9cd0d3d24772132febf52",
		Name:     "Dr. Hofstadter",
		Email:    "leanard@caltech.edu",
		Gravatar: "23dde632fdece6880f4ff03bb20f05d7",
		Token:    "a5ad0d75f317f0b0a5dfdb68e5a3079e",
		Admin:    true,
		Active:   true,
		Syncing:  false,
		Created:  1398065343,
		Updated:  1398065344,
		Synced:   1398065345,
	})

	meddler.Insert(db, "users", &model.User{
		Remote:   "gitlab.com",
		Login:    "browndynamite",
		Access:   "4821477cc26a0c8c80c6c9b568d98e32",
		Secret:   "1dd52c37cf5c63fe5abfd047b5b74a31",
		Name:     "Dr. Koothrappali",
		Email:    "rajesh@caltech.edu",
		Gravatar: "f9133051f480b7ea88848b9f0a079dae",
		Token:    "7a50ede04637d4a8fce532c7d511226b",
		Admin:    true,
		Active:   true,
		Syncing:  false,
		Created:  1398065343,
		Updated:  1398065344,
		Synced:   1398065345,
	})

	meddler.Insert(db, "users", &model.User{
		Remote:   "github.com",
		Login:    "mrwolowitz",
		Access:   "1f6a80bde960e6913bf9b7e61eadd068",
		Secret:   "74c40472494ba7f9f6c3ae061ff799ed",
		Name:     "Mr. Wolowitz",
		Email:    "wolowitz@caltech.edu",
		Gravatar: "ea250570c794d84dc583421bb717be82",
		Token:    "3bd7e7d7411b2978e45919c9ad419984",
		Admin:    true,
		Active:   true,
		Syncing:  false,
		Created:  1398065343,
		Updated:  1398065344,
		Synced:   1398065345,
	})
}
