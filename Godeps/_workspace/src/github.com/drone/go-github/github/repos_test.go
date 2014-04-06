package github

import (
	"testing"
)

func Test_Repos(t *testing.T) {

	repos, err := client.Repos.List()
	if err != nil {
		t.Error(err)
		return
	}
	if len(repos) == 0 {
		t.Errorf("List of repos returned empty set")
	}

	// Get the named repo
	repo, err := client.Repos.Find(testUser, testRepo)
	if err != nil {
		t.Error(err)
		return
	}
	if repo.Name != testRepo {
		t.Errorf("repo name [%v]; want [%v]", repo.Name, testRepo)
	}

	// Get ALL repos, including team repos
	repos, err = client.Repos.ListAll()
	if err != nil {
		t.Error(err)
		return
	}
	if len(repos) == 0 {
		t.Errorf("List of ALL repos returned empty set")
	}
}

func Test_ReposFind(t *testing.T) {
	client.Repos.Find(testUser, testRepo)
}

func Test_ReposListOrgs(t *testing.T) {
	repos, err := client.Repos.ListOrg("drone")
	if err != nil {
		t.Error(err)
		return
	}
	if len(repos) == 0 {
		t.Errorf("List of repos returned empty set")
	}
}
