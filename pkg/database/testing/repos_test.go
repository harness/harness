package database

import (
	"testing"

	"github.com/drone/drone/pkg/database"
)

// TODO unit test to verify unique constraint on Member.UserID and Member.TeamID

// TestGetRepo tests the ability to retrieve a Repo
// from the database by Unique ID.
func TestGetRepo(t *testing.T) {
	Setup()
	defer Teardown()

	repo, err := database.GetRepo(1)
	if err != nil {
		t.Error(err)
	}

	if repo.ID != 1 {
		t.Errorf("Exepected ID %d, got %d", 1, repo.ID)
	}

	if repo.Slug != "github.com/drone/drone" {
		t.Errorf("Exepected Slug %s, got %s", "github.com/drone/drone", repo.Slug)
	}

	if repo.Host != "github.com" {
		t.Errorf("Exepected Host %s, got %s", "github.com", repo.Host)
	}

	if repo.Owner != "drone" {
		t.Errorf("Exepected Owner %s, got %s", "drone", repo.Owner)
	}

	if repo.Name != "drone" {
		t.Errorf("Exepected Name %s, got %s", "drone", repo.Name)
	}

	if repo.Private != true {
		t.Errorf("Exepected Private %v, got %v", true, repo.Private)
	}

	if repo.Disabled != false {
		t.Errorf("Exepected Private %v, got %v", false, repo.Disabled)
	}

	if repo.SCM != "git" {
		t.Errorf("Exepected Type %s, got %s", "git", repo.SCM)
	}

	if repo.URL != "git@github.com:drone/drone.git" {
		t.Errorf("Exepected URL %s, got %s", "git@github.com:drone/drone.git", repo.URL)
	}

	if repo.Username != "no username" {
		t.Errorf("Exepected Username %s, got %s", "no username", repo.Username)
	}

	if repo.Password != "no password" {
		t.Errorf("Exepected Password %s, got %s", "no password", repo.Password)
	}

	if repo.PublicKey != pubkey {
		t.Errorf("Exepected PublicKey %s, got %s", "public key", repo.PublicKey)
	}

	if repo.PrivateKey != privkey {
		t.Errorf("Exepected PrivateKey %s, got %s", "private key", repo.PrivateKey)
	}

	if repo.UserID != 1 {
		t.Errorf("Exepected ID %d, got %d", 1, repo.UserID)
	}

	if repo.TeamID != 1 {
		t.Errorf("Exepected ID %d, got %d", 1, repo.TeamID)
	}
}

// TestGetRepoSlug tests the ability to retrieve a Repo
// from the database by it's Canonical Name.
func TestGetRepoSlug(t *testing.T) {
	Setup()
	defer Teardown()

	repo, err := database.GetRepoSlug("github.com/drone/drone")
	if err != nil {
		t.Error(err)
	}

	if repo.ID != 1 {
		t.Errorf("Exepected ID %d, got %d", 1, repo.ID)
	}

	if repo.Slug != "github.com/drone/drone" {
		t.Errorf("Exepected Slug %s, got %s", "github.com/drone/drone", repo.Slug)
	}

	if repo.Host != "github.com" {
		t.Errorf("Exepected Host %s, got %s", "github.com", repo.Host)
	}

	if repo.Owner != "drone" {
		t.Errorf("Exepected Owner %s, got %s", "drone", repo.Owner)
	}

	if repo.Name != "drone" {
		t.Errorf("Exepected Name %s, got %s", "drone", repo.Name)
	}

	if repo.Private != true {
		t.Errorf("Exepected Private %v, got %v", true, repo.Private)
	}

	if repo.Disabled != false {
		t.Errorf("Exepected Private %v, got %v", false, repo.Disabled)
	}

	if repo.SCM != "git" {
		t.Errorf("Exepected Type %s, got %s", "git", repo.SCM)
	}

	if repo.URL != "git@github.com:drone/drone.git" {
		t.Errorf("Exepected URL %s, got %s", "git@github.com:drone/drone.git", repo.URL)
	}

	if repo.Username != "no username" {
		t.Errorf("Exepected Username %s, got %s", "no username", repo.Username)
	}

	if repo.Password != "no password" {
		t.Errorf("Exepected Password %s, got %s", "no password", repo.Password)
	}

	if repo.PublicKey != pubkey {
		t.Errorf("Exepected PublicKey %s, got %s", "public key", repo.PublicKey)
	}

	if repo.PrivateKey != privkey {
		t.Errorf("Exepected PrivateKey %s, got %s", "private key", repo.PrivateKey)
	}

	if repo.UserID != 1 {
		t.Errorf("Exepected ID %d, got %d", 1, repo.UserID)
	}

	if repo.TeamID != 1 {
		t.Errorf("Exepected ID %d, got %d", 1, repo.TeamID)
	}
}

func TestSaveRepo(t *testing.T) {
	Setup()
	defer Teardown()

	// get the repo we plan to update
	repo, err := database.GetRepo(1)
	if err != nil {
		t.Error(err)
	}

	// update fields
	repo.Slug = "bitbucket.org/drone/drone"
	repo.Host = "bitbucket.org"
	repo.Private = false
	repo.Disabled = true
	repo.SCM = "hg"
	repo.URL = "https://bitbucket.org/drone/drone"
	repo.Username = "brad"
	repo.Password = "password"
	repo.TeamID = 0

	// update the database
	if err := database.SaveRepo(repo); err != nil {
		t.Error(err)
	}

	// get the updated repo
	updatedRepo, err := database.GetRepo(1)
	if err != nil {
		t.Error(err)
	}

	if updatedRepo.Slug != repo.Slug {
		t.Errorf("Exepected Slug %s, got %s", updatedRepo.Slug, repo.Slug)
	}

	if updatedRepo.Host != repo.Host {
		t.Errorf("Exepected Host %s, got %s", updatedRepo.Host, repo.Host)
	}

	if updatedRepo.Private != repo.Private {
		t.Errorf("Exepected Private %v, got %v", updatedRepo.Private, repo.Private)
	}

	if updatedRepo.Disabled != repo.Disabled {
		t.Errorf("Exepected Private %v, got %v", updatedRepo.Disabled, repo.Disabled)
	}

	if updatedRepo.SCM != repo.SCM {
		t.Errorf("Exepected Type %s, got %s", true, repo.SCM)
	}

	if updatedRepo.URL != repo.URL {
		t.Errorf("Exepected URL %s, got %s", updatedRepo.URL, repo.URL)
	}

	if updatedRepo.Username != repo.Username {
		t.Errorf("Exepected Username %s, got %s", updatedRepo.Username, repo.Username)
	}

	if updatedRepo.Password != repo.Password {
		t.Errorf("Exepected Password %s, got %s", updatedRepo.Password, repo.Password)
	}

	if updatedRepo.TeamID != repo.TeamID {
		t.Errorf("Exepected TeamID %d, got %d", updatedRepo.TeamID, repo.TeamID)
	}
}

func TestDeleteRepo(t *testing.T) {
	Setup()
	defer Teardown()

	if err := database.DeleteRepo(1); err != nil {
		t.Error(err)
	}

	// try to get the deleted row
	_, err := database.GetRepo(1)
	if err == nil {
		t.Fail()
	}
}

/*
func TestListRepos(t *testing.T) {
	Setup()
	defer Teardown()

	// repos for user_id = 1
	repos, err := database.ListRepos(1)
	if err != nil {
		t.Error(err)
	}

	// verify user count
	if len(repos) != 2 {
		t.Errorf("Exepected %d repos in database, got %d", 2, len(repos))
		return
	}

	// get the second repo in the list and verify
	// fields are being populated correctly
	// NOTE: we get the 2nd repo due to sorting
	repo := repos[1]

	if repo.ID != 1 {
		t.Errorf("Exepected ID %d, got %d", 1, repo.ID)
	}

	if repo.Name != "github.com/drone/drone" {
		t.Errorf("Exepected Name %s, got %s", "github.com/drone/drone", repo.Name)
	}

	if repo.Host != "github.com" {
		t.Errorf("Exepected Host %s, got %s", "github.com", repo.Host)
	}

	if repo.Owner != "drone" {
		t.Errorf("Exepected Owner %s, got %s", "drone", repo.Owner)
	}

	if repo.Slug != "drone" {
		t.Errorf("Exepected Slug %s, got %s", "drone", repo.Slug)
	}

	if repo.Private != true {
		t.Errorf("Exepected Private %v, got %v", true, repo.Private)
	}

	if repo.Disabled != false {
		t.Errorf("Exepected Private %v, got %v", false, repo.Disabled)
	}

	if repo.SCM != "git" {
		t.Errorf("Exepected Type %s, got %s", "git", repo.SCM)
	}

	if repo.URL != "git@github.com:drone/drone.git" {
		t.Errorf("Exepected URL %s, got %s", "git@github.com:drone/drone.git", repo.URL)
	}

	if repo.Username != "no username" {
		t.Errorf("Exepected Username %s, got %s", "no username", repo.Username)
	}

	if repo.Password != "no password" {
		t.Errorf("Exepected Password %s, got %s", "no password", repo.Password)
	}

	if repo.PublicKey != "public key" {
		t.Errorf("Exepected PublicKey %s, got %s", "public key", repo.PublicKey)
	}

	if repo.PrivateKey != "private key" {
		t.Errorf("Exepected PrivateKey %s, got %s", "private key", repo.PrivateKey)
	}

	if repo.UserID != 1 {
		t.Errorf("Exepected ID %d, got %d", 1, repo.UserID)
	}

	if repo.TeamID != 1 {
		t.Errorf("Exepected ID %d, got %d", 1, repo.TeamID)
	}
}
*/

func TestListReposTeam(t *testing.T) {
	Setup()
	defer Teardown()

	// repos for team_id = 1
	repos, err := database.ListReposTeam(1)
	if err != nil {
		t.Error(err)
	}

	// verify user count
	if len(repos) != 2 {
		t.Errorf("Exepected %d repos in database, got %d", 2, len(repos))
		return
	}

	// get the second repo in the list and verify
	// fields are being populated correctly
	// NOTE: we get the 2nd repo due to sorting
	repo := repos[1]

	if repo.ID != 1 {
		t.Errorf("Exepected ID %d, got %d", 1, repo.ID)
	}

	if repo.Slug != "github.com/drone/drone" {
		t.Errorf("Exepected Slug %s, got %s", "github.com/drone/drone", repo.Slug)
	}

	if repo.Host != "github.com" {
		t.Errorf("Exepected Host %s, got %s", "github.com", repo.Host)
	}

	if repo.Owner != "drone" {
		t.Errorf("Exepected Owner %s, got %s", "drone", repo.Owner)
	}

	if repo.Name != "drone" {
		t.Errorf("Exepected Name %s, got %s", "drone", repo.Name)
	}

	if repo.Private != true {
		t.Errorf("Exepected Private %v, got %v", true, repo.Private)
	}

	if repo.Disabled != false {
		t.Errorf("Exepected Private %v, got %v", false, repo.Disabled)
	}

	if repo.SCM != "git" {
		t.Errorf("Exepected Type %s, got %s", "git", repo.SCM)
	}

	if repo.URL != "git@github.com:drone/drone.git" {
		t.Errorf("Exepected URL %s, got %s", "git@github.com:drone/drone.git", repo.URL)
	}

	if repo.Username != "no username" {
		t.Errorf("Exepected Username %s, got %s", "no username", repo.Username)
	}

	if repo.Password != "no password" {
		t.Errorf("Exepected Password %s, got %s", "no password", repo.Password)
	}

	if repo.PublicKey != pubkey {
		t.Errorf("Exepected PublicKey %s, got %s", "public key", repo.PublicKey)
	}

	if repo.PrivateKey != privkey {
		t.Errorf("Exepected PrivateKey %s, got %s", "private key", repo.PrivateKey)
	}

	if repo.UserID != 1 {
		t.Errorf("Exepected ID %d, got %d", 1, repo.UserID)
	}

	if repo.TeamID != 1 {
		t.Errorf("Exepected ID %d, got %d", 1, repo.TeamID)
	}
}
