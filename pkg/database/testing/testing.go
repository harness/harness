package database

import (
	"crypto/aes"
	"database/sql"
	"log"

	"github.com/drone/drone/pkg/database"
	"github.com/drone/drone/pkg/database/encrypt"
	"github.com/drone/drone/pkg/database/migrate"
	. "github.com/drone/drone/pkg/model"

	_ "github.com/mattn/go-sqlite3"
	"github.com/russross/meddler"
)

// in-memory database used for
// unit testing purposes.
var db *sql.DB

func init() {
	// create a cipher for ecnrypting and decrypting
	// database fields
	cipher, err := aes.NewCipher([]byte("38B241096B8DA08131563770F4CDDFAC"))
	if err != nil {
		log.Fatal(err)
	}

	// register function with meddler to encrypt and
	// decrypt database fields.
	meddler.Register("gobencrypt", &encrypt.EncryptedField{cipher})

	// notify meddler that we are working with sqlite
	meddler.Default = meddler.SQLite
	migrate.Driver = migrate.SQLite
}

func Setup() {
	// create an in-memory database
	db, _ = sql.Open("sqlite3", ":memory:")

	// make sure all the tables and indexes are created
	database.Set(db)

	migration := migrate.New(db)
	migration.All().Migrate()

	// create dummy user data
	user1 := User{
		Password: "$2a$10$b8d63QsTL38vx7lj0HEHfOdbu1PCAg6Gfca74UavkXooIBx9YxopS",
		Name:     "Brad Rydzewski",
		Email:    "brad.rydzewski@gmail.com",
		Gravatar: "8c58a0be77ee441bb8f8595b7f1b4e87",
		Token:    "123",
		Admin:    true}
	user2 := User{
		Password: "$2a$10$b8d63QsTL38vx7lj0HEHfOdbu1PCAg6Gfca74UavkXooIBx9YxopS",
		Name:     "Thomas Burke",
		Email:    "cavepig@gmail.com",
		Gravatar: "c62f7126273f7fa786274274a5dec8ce",
		Token:    "456",
		Admin:    false}
	user3 := User{
		Password: "$2a$10$b8d63QsTL38vx7lj0HEHfOdbu1PCAg6Gfca74UavkXooIBx9YxopS",
		Name:     "Carlos Morales",
		Email:    "ytsejammer@gmail.com",
		Gravatar: "c2180a539620d90d68eaeb848364f1c2",
		Token:    "789",
		Admin:    false}

	database.SaveUser(&user1)
	database.SaveUser(&user2)
	database.SaveUser(&user3)

	// create dummy team data
	team1 := Team{
		Slug:     "drone",
		Name:     "Drone",
		Email:    "support@drone.io",
		Gravatar: "8c58a0be77ee441bb8f8595b7f1b4e87"}
	team2 := Team{
		Slug:     "github",
		Name:     "Github",
		Email:    "support@github.com",
		Gravatar: "61024896f291303615bcd4f7a0dcfb74"}
	team3 := Team{
		Slug:     "golang",
		Name:     "Golang",
		Email:    "support@golang.org",
		Gravatar: "991695cc770c6b8354b68cd18c280b95"}

	database.SaveTeam(&team1)
	database.SaveTeam(&team2)
	database.SaveTeam(&team3)

	// create team membership data
	database.SaveMember(user1.ID, team1.ID, RoleOwner)
	database.SaveMember(user2.ID, team1.ID, RoleAdmin)
	database.SaveMember(user3.ID, team1.ID, RoleWrite)
	database.SaveMember(user1.ID, team2.ID, RoleOwner)
	database.SaveMember(user2.ID, team2.ID, RoleAdmin)
	database.SaveMember(user3.ID, team2.ID, RoleWrite)
	database.SaveMember(user1.ID, team3.ID, RoleRead)

	// create dummy repo data
	repo1 := Repo{
		Slug:       "github.com/drone/drone",
		Host:       "github.com",
		Owner:      "drone",
		Name:       "drone",
		Private:    true,
		Disabled:   false,
		SCM:        "git",
		URL:        "git@github.com:drone/drone.git",
		Username:   "no username",
		Password:   "no password",
		PublicKey:  "public key",
		PrivateKey: "private key",
		UserID:     user1.ID,
		TeamID:     team1.ID,
	}
	repo2 := Repo{
		Slug:       "bitbucket.org/drone/test",
		Host:       "bitbucket.org",
		Owner:      "drone",
		Name:       "test",
		Private:    false,
		Disabled:   false,
		SCM:        "hg",
		URL:        "https://bitbucket.org/drone/test",
		Username:   "no username",
		Password:   "no password",
		PublicKey:  "public key",
		PrivateKey: "private key",
		UserID:     user1.ID,
		TeamID:     team1.ID,
	}
	repo3 := Repo{
		Slug:       "bitbucket.org/brydzewski/test",
		Host:       "bitbucket.org",
		Owner:      "brydzewski",
		Name:       "test",
		Private:    false,
		Disabled:   false,
		SCM:        "hg",
		URL:        "https://bitbucket.org/brydzewski/test",
		Username:   "no username",
		Password:   "no password",
		PublicKey:  "public key",
		PrivateKey: "private key",
		UserID:     user2.ID,
	}

	database.SaveRepo(&repo1)
	database.SaveRepo(&repo2)
	database.SaveRepo(&repo3)

	commit1 := Commit{
		RepoID:   repo1.ID,
		Status:   "Success",
		Hash:     "4f4c4594be6d6ddbc1c0dd521334f7ecba92b608",
		Branch:   "master",
		Author:   user1.Email,
		Gravatar: user1.Gravatar,
		Message:  "commit message",
	}
	commit2 := Commit{
		RepoID:   repo1.ID,
		Status:   "Failure",
		Hash:     "0eb2fa13e9f4139e803b6ad37831708d4786c74a",
		Branch:   "master",
		Author:   user1.Email,
		Gravatar: user1.Gravatar,
		Message:  "commit message",
	}
	commit3 := Commit{
		RepoID:   repo1.ID,
		Status:   "Failure",
		Hash:     "60a7fe87ccf01d0152e53242528399e05acaf047",
		Branch:   "dev",
		Author:   user1.Email,
		Gravatar: user1.Gravatar,
		Message:  "commit message",
	}
	commit4 := Commit{
		RepoID:   repo2.ID,
		Status:   "Success",
		Hash:     "a4078d1e9a0842cdd214adbf0512578799a4f2ba",
		Branch:   "master",
		Author:   user1.Email,
		Gravatar: user1.Gravatar,
		Message:  "commit message",
	}

	// create dummy commit data
	database.SaveCommit(&commit1)
	database.SaveCommit(&commit2)
	database.SaveCommit(&commit3)
	database.SaveCommit(&commit4)

	// create dummy build data
	database.SaveBuild(&Build{CommitID: commit1.ID, Slug: "node_0.10", Status: "Success", Duration: 60})
	database.SaveBuild(&Build{CommitID: commit1.ID, Slug: "node_0.09", Status: "Success", Duration: 70})
	database.SaveBuild(&Build{CommitID: commit2.ID, Slug: "node_0.10", Status: "Success", Duration: 10})
	database.SaveBuild(&Build{CommitID: commit2.ID, Slug: "node_0.09", Status: "Failure", Duration: 65})
	database.SaveBuild(&Build{CommitID: commit3.ID, Slug: "node_0.10", Status: "Failure", Duration: 50})
	database.SaveBuild(&Build{CommitID: commit3.ID, Slug: "node_0.09", Status: "Failure", Duration: 55})
}

func Teardown() {
	db.Close()
}
