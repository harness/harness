package condition

import (
	"testing"
)

type Bool bool

func Test_MatchPullRequest(t *testing.T) {

	var c = Condition{}
	var got, want = c.MatchPullRequest(""), true
	if got != want {
		t.Errorf("Non-pull requests are always enabled, expected %v, got %v", want, got)
	}

	got, want = c.MatchPullRequest("65"), false
	if got != want {
		t.Errorf("Pull requests should be disabled by default, expected %v, got %v", want, got)
	}

	c.PullRequest = new(bool)
	*c.PullRequest = false
	got, want = c.MatchPullRequest("65"), false
	if got != want {
		t.Errorf("Pull requests can be explicity disabled, expected %v, got %v", want, got)
	}

	c.PullRequest = new(bool)
	*c.PullRequest = true
	got, want = c.MatchPullRequest("65"), true
	if got != want {
		t.Errorf("Pull requests can be explicitly enabled, expected %v, got %v", want, got)
	}
}

func Test_MatchBranch(t *testing.T) {

	var c = Condition{}
	var got, want = c.MatchBranch("master"), true
	if got != want {
		t.Errorf("All branches should be enabled by default, expected %v, got %v", want, got)
	}

	c.Branch = ""
	got, want = c.MatchBranch("master"), true
	if got != want {
		t.Errorf("Empty branch should match, expected %v, got %v", want, got)
	}

	c.Branch = "master"
	got, want = c.MatchBranch("master"), true
	if got != want {
		t.Errorf("Branch should match, expected %v, got %v", want, got)
	}

	c.Branch = "master"
	got, want = c.MatchBranch("dev"), false
	if got != want {
		t.Errorf("Branch should not match, expected %v, got %v", want, got)
	}

	c.Branch = "release/*"
	got, want = c.MatchBranch("release/1.0.0"), true
	if got != want {
		t.Errorf("Branch should match wildcard, expected %v, got %v", want, got)
	}
}

func Test_MatchOwner(t *testing.T) {

	var c = Condition{}
	var got, want = c.MatchOwner("drone"), true
	if got != want {
		t.Errorf("All owners should be enabled by default, expected %v, got %v", want, got)
	}

	c.Owner = ""
	got, want = c.MatchOwner("drone"), true
	if got != want {
		t.Errorf("Empty owner should match, expected %v, got %v", want, got)
	}

	c.Owner = "drone"
	got, want = c.MatchOwner("drone"), true
	if got != want {
		t.Errorf("Owner should match, expected %v, got %v", want, got)
	}

	c.Owner = "drone"
	got, want = c.MatchOwner("drone/config"), true
	if got != want {
		t.Errorf("Owner/Repo should match, expected %v, got %v", want, got)
	}

	c.Owner = "drone"
	got, want = c.MatchOwner("github.com/drone/config"), true
	if got != want {
		t.Errorf("Host/Owner/Repo should match, expected %v, got %v", want, got)
	}

	c.Owner = "bradrydzewski"
	got, want = c.MatchOwner("drone"), false
	if got != want {
		t.Errorf("Owner should not match, expected %v, got %v", want, got)
	}

	c.Owner = "drone"
	got, want = c.MatchOwner("bradrydzewski/drone"), false
	if got != want {
		t.Errorf("Owner/Repo should not match, expected %v, got %v", want, got)
	}

	c.Owner = "drone"
	got, want = c.MatchOwner("github.com/bradrydzewski/drone"), false
	if got != want {
		t.Errorf("Host/Owner/Repo should not match, expected %v, got %v", want, got)
	}
}
