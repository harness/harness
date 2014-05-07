package database

import (
	"crypto/aes"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/drone/drone/pkg/database"
	"github.com/drone/drone/pkg/database/encrypt"
	. "github.com/drone/drone/pkg/model"

	_ "github.com/mattn/go-sqlite3"
	"github.com/russross/meddler"
)

const (
	pubkey  = `sh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQClp9+xjhYj2Wz0nwLNhUiR1RkqfoVZwlJoxdubQy8GskZtY7C7YGa/PeKfdfaKOWtVgg37r/OYS3kc7bIKVup4sx/oW59FMwCZYQ2nxoaPZpPwUJs8D0Wy0b2VSP+vAnJ6jZQEIEiClrzyYafSfqN6L9T/BTkn28ktWalOHqWVKejKeD6M0uhlpyIZFsQ1K2wNt32ACwT/rbanx/r/jfczqxSkLzvIKXXs/RdKQgwRRUYnKkl4Lh6r22n9n3m2VwRor5wdsPK8sr57OsqdRpnvsFs3lxwM8w5ZiAZV3T0xTMGVs3W8Uy5HexAD6TgWBWFjSrgdXF1pE83wmUtJtVBf`
	privkey = `-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEApaffsY4WI9ls9J8CzYVIkdUZKn6FWcJSaMXbm0MvBrJGbWOw
u2Bmvz3in3X2ijlrVYIN+6/zmEt5HO2yClbqeLMf6FufRTMAmWENp8aGj2aT8FCb
PA9FstG9lUj/rwJyeo2UBCBIgpa88mGn0n6jei/U/wU5J9vJLVmpTh6llSnoyng+
jNLoZaciGRbENStsDbd9gAsE/622p8f6/433M6sUpC87yCl17P0XSkIMEUVGJypJ
eC4eq9tp/Z95tlcEaK+cHbDyvLK+ezrKnUaZ77BbN5ccDPMOWYgGVd09MUzBlbN1
vFMuR3sQA+k4FgVhY0q4HVxdaRPN8JlLSbVQXwIDAQABAoIBAA3EqSPxwkdSf+rI
+IuqY0CzrHbKszylmQHaSAlciSEOWionWf4I4iFM/HPycv5EDXa663yawC1NQJC1
9NFFLhHAGYvPaapvtcIJvf/O0UpD5VHY8T4JqupU4mPxAEdEdc1XzRCWulAYRTYE
BdXJ7r5uEU7s2TZF3y+kvxyeEXcXNWK1I4kGBSgH4KI5WIODtNJ6vaIk5Yugqt1N
cg5Sprk4bUMRTBH6GmSiJUleA0f/k6MCCmhETKXGt9mmfJ1PXpVlfDn5m26MX6vZ
XgaoIHUCy4sh1Fq6vbEI831JcO4kdvl4TtX90SzSadHjewNHy0V2gjAysvqbEDhw
Hn8D+MkCgYEA00tTKPp3AUTxT9ZgfPBD3DY7tk7+xp2R2lA6H9TUWe79TRfncFtS
8bCfEXd8xALL5cFyzi4q4YJ77mJjlWE7AMYdsFoAW1l3Q71JRRBSwsyIwp4hU8AV
K48SDjqecDzY42UvuKGp3opPWb0PzJixJNUgawU/ZGPxqN8jlr0o+K0CgYEAyLSO
rZqOvyE5wu8yadHLlQC4usoYtnyDC8PG2SgnZnlnZnkgNy3yLmHYvTvYSQsAv7rA
fFsKMt2MJhlclx+sTds/LLHKj/RfVDFenFf6ajBNZ1k+KRcwrV1A4iWinWmBxiEi
A8aM9rGs7WRBkqaCONSUQHcmLRRz7hqDtsBpkrsCgYBY2FJ2Z6LEmN2zCVx3DHws
S22eQeclUroyhwt5uP81daVy1jtN5kihMfgg2xJORTLBQC9q/MSxIDHGUf63oDO0
JpnzPlTqFFtu01fMv4ldOa3Dz8QJuDnun/EipIlcfmlgbHq9ctS/q36kKDhNemL6
Lte7yHAYYWIK9RC84Hsq3QKBgAfDbC1s6A6ek2Rl6jZLpitKTtryzEfqwwrmdL+b
nQKKuaQuFT/tKAwBPuf685/HrCy+ZYmp39gd17j1jC5QTFLqoyPwcJxm4HUaP8We
ZZJL8gKIYi4mtnxOOh9FQ2gBV8K5L16kBHnaX40DLsIkbK8UEfP4Z+Kggud34RZl
lO/XAoGAFFZdolsVbSieFhJt7ypzp/19dKJ8Sk6QGCk3uQpTuLJMvwcBT8X5XCTD
zFfYARarx87mbD2k5GZ7F0fmGYTUV14qlxJCGMythLM/xZ6EJuJWBz69puNj4yhn
exWM7t1BDHy2zIoPfIQLDH2h1zNTRjismMeErOCy0Uha7jrZhW8=
-----END RSA PRIVATE KEY-----`
)

var (
	dbname, driver, dsn, login string
	db                         *sql.DB
)

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

	// Check for $DB_ENV
	dbenv := os.Getenv("DB_ENV")
	if dbenv == "mysql" {
		driver = dbenv
		dbname = "drone_test"
		login = os.Getenv("MYSQL_LOGIN")
		if len(login) == 0 {
			login = "root"
		}
		log.Println("Using mysql database ...")
	} else {
		driver = "sqlite3"
		dsn = ":memory:"
		log.Println("Using sqlite3 database ...")
	}

}

func Setup() {
	// create an in-memory database
	if driver == "mysql" {
		idsn := fmt.Sprintf("%s@/?parseTime=true", login)
		db, dsn = createDB(dbname, idsn)
	}
	database.Init(driver, dsn)

	// create dummy user data
	user1 := User{
		Password:    "$2a$10$b8d63QsTL38vx7lj0HEHfOdbu1PCAg6Gfca74UavkXooIBx9YxopS",
		Name:        "Brad Rydzewski",
		Email:       "brad.rydzewski@gmail.com",
		Gravatar:    "8c58a0be77ee441bb8f8595b7f1b4e87",
		Token:       "123",
		GitlabToken: "123",
		Admin:       true}
	user2 := User{
		Password:    "$2a$10$b8d63QsTL38vx7lj0HEHfOdbu1PCAg6Gfca74UavkXooIBx9YxopS",
		Name:        "Thomas Burke",
		Email:       "cavepig@gmail.com",
		Gravatar:    "c62f7126273f7fa786274274a5dec8ce",
		Token:       "456",
		GitlabToken: "456",
		Admin:       false}
	user3 := User{
		Password:    "$2a$10$b8d63QsTL38vx7lj0HEHfOdbu1PCAg6Gfca74UavkXooIBx9YxopS",
		Name:        "Carlos Morales",
		Email:       "ytsejammer@gmail.com",
		Gravatar:    "c2180a539620d90d68eaeb848364f1c2",
		Token:       "789",
		GitlabToken: "789",
		Admin:       false}
	user4 := User{
		Password:    "$2a$10$b8d63QsTL38vx7lj0HEHfOdbu1PCAg6Gfca74UavkXooIBx9YxopS",
		Name:        "Rick El Toro",
		Email:       "rick@el.to.ro",
		Gravatar:    "c2180a539620d90d68eaeb848364f1c2",
		Token:       "987",
		GitlabToken: "987",
		Admin:       false}


	database.SaveUser(&user1)
	database.SaveUser(&user2)
	database.SaveUser(&user3)
	database.SaveUser(&user4)

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
		PublicKey:  pubkey,
		PrivateKey: privkey,
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
		PublicKey:  pubkey,
		PrivateKey: privkey,
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
		PublicKey:  pubkey,
		PrivateKey: privkey,
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
	commit5 := Commit{
		RepoID:   repo2.ID,
		Status:   "Success",
		Hash:     "5f32ec7b08dfe3a097c1a5316de5b5069fb35ff9",
		Branch:   "develop",
		Author:   user1.Email,
		Gravatar: user1.Gravatar,
		Message:  "commit message",
	}

	// create dummy commit data
	database.SaveCommit(&commit1)
	database.SaveCommit(&commit2)
	database.SaveCommit(&commit3)
	database.SaveCommit(&commit4)
	database.SaveCommit(&commit5)

	// create dummy build data
	database.SaveBuild(&Build{CommitID: commit1.ID, Slug: "node_0.10", Status: "Success", Duration: 60})
	database.SaveBuild(&Build{CommitID: commit1.ID, Slug: "node_0.09", Status: "Success", Duration: 70})
	database.SaveBuild(&Build{CommitID: commit2.ID, Slug: "node_0.10", Status: "Success", Duration: 10})
	database.SaveBuild(&Build{CommitID: commit2.ID, Slug: "node_0.09", Status: "Failure", Duration: 65})
	database.SaveBuild(&Build{CommitID: commit3.ID, Slug: "node_0.10", Status: "Failure", Duration: 50})
	database.SaveBuild(&Build{CommitID: commit3.ID, Slug: "node_0.09", Status: "Failure", Duration: 55})
}

func Teardown() {
	database.Close()
	if driver == "mysql" {
		db.Exec(fmt.Sprintf("DROP DATABASE %s", dbname))
	}
}

func createDB(name, datasource string) (*sql.DB, string) {
	db, err := sql.Open(driver, datasource)
	if err != nil {
		panic("Can't connect to database")
	}
	if _, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s", name)); err != nil {
		panic("Can't create database")
	}
	dsn := strings.Replace(datasource, "/", fmt.Sprintf("/%s", name), 1)
	return db, dsn
}
