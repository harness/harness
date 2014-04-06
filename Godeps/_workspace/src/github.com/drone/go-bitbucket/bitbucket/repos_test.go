package bitbucket

import (
	"testing"
)

func Test_Repos(t *testing.T) {

	// LIST of repositories
	repos, err := client.Repos.List()
	if err != nil {
		t.Error(err)
	}

	if len(repos) == 0 {
		t.Errorf("List of /user repositories returned empty set")
	}
	
	// LIST dashboard repositories
	accts, err := client.Repos.ListDashboard()
	if err != nil {
		t.Error(err)
	}

	if len(accts) == 0 {
		t.Errorf("List of dashboard repositories returned empty set")
	}
	
	// FIND the named repo
	repo, err := client.Repos.Find(testUser, testRepo)
	if err != nil {
		t.Error(err)
	}

	if repo.Slug != testRepo {
		t.Errorf("repo slug [%v]; want [%v]", repo.Slug, testRepo)
	}
}
