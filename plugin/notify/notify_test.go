package notify

import (
	"testing"

	"github.com/drone/drone/shared/model"
)

func Test_getBuildUrl(t *testing.T) {
	c := &model.Request{
		Host: "http://examplehost.com",
		Repo: &model.Repo{
			Host:  "examplegit.com",
			Owner: "owner",
			Name:  "repo",
		},
		Commit: &model.Commit{
			Sha:    "abc",
			Branch: "example",
		},
	}
	expected := "http://examplehost.com/examplegit.com/owner/repo/example/abc"
	output := getBuildUrl(c)

	if output != expected {
		t.Errorf("Failed to build url. Expected: %s, got %s", expected, output)
	}
}
