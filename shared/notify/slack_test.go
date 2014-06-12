package notify

import (
	"github.com/drone/drone/server/resource/commit"
	"github.com/drone/drone/server/resource/repo"
	"testing"
)

func Test_getBuildUrl(t *testing.T) {
	c := &Context{
		Host: "http://examplehost.com",
		Repo: &repo.Repo{
			Host:  "examplegit.com",
			Owner: "owner",
			Name:  "repo",
		},
		Commit: &commit.Commit{
			Sha:    "abc",
			Branch: "example",
		},
	}
	expected := "http://examplehost.com/examplegit.com/owner/repo/branch/example/commit/abc"
	output := getBuildUrl(c)

	if output != expected {
		t.Errorf("Failed to build url. Expected: %s, got %s", expected, output)
	}
}
