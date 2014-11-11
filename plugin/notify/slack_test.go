package notify

import "testing"

/*
var request = &model.Request{
	Host: "http://examplehost.com",
	Repo: &model.Repo{
		Host:  "examplegit.com",
		Owner: "owner",
		Name:  "repo",
	},
	Commit: &model.Commit{
		Sha:     "abc",
		Branch:  "example",
		Status:  "Started",
		Message: "Test Commit",
		Author:  "Test User",
	},
	User: &model.User{
		Login: "TestUser",
	},
}
*/

var (
	slackExpectedLink         = "<http://examplehost.com/examplegit.com/owner/repo/example/abc|owner/repo#abc>"
	slackExpectedFallbackText = "owner/repo#abc (example) by Test User"
	slackExpectedBase         = slackExpectedLink + " (example) by Test User"
)

func Test_slackStartedMessage(t *testing.T) {
	actual := (&Slack{}).getMessage(request, slackStartedMessage)

	expected := "*Building* " + slackExpectedBase

	if actual != expected {
		t.Errorf("Invalid getStarted message for Slack. Expected %v, got %v", expected, actual)
	}
}

func Test_slackStartedFallbackMessage(t *testing.T) {
	actual := (&Slack{}).getFallbackMessage(request, slackStartedFallbackMessage)

	expected := "Building " + slackExpectedFallbackText

	if actual != expected {
		t.Errorf("Invalid fallback started message for Slack. Expected %v, got %v", expected, actual)
	}
}

func Test_slackSuccessMessage(t *testing.T) {
	actual := (&Slack{}).getMessage(request, slackSuccessMessage)

	expected := "*Success* " + slackExpectedBase

	if actual != expected {
		t.Errorf("Invalid getStarted message for Slack. Expected %v, got %v", expected, actual)
	}
}

func Test_slackSuccessFallbackMessage(t *testing.T) {
	actual := (&Slack{}).getFallbackMessage(request, slackSuccessFallbackMessage)

	expected := "Success " + slackExpectedFallbackText

	if actual != expected {
		t.Errorf("Invalid success fallback message for Slack. Expected %v, got %v", expected, actual)
	}
}

func Test_slackFailureMessage(t *testing.T) {
	actual := (&Slack{}).getMessage(request, slackFailureMessage)

	expected := "*Failed* " + slackExpectedBase

	if actual != expected {
		t.Errorf("Invalid getStarted message for Slack. Expected %v, got %v", expected, actual)
	}
}

func Test_slackFailureFallbackMessage(t *testing.T) {
	actual := (&Slack{}).getFallbackMessage(request, slackFailureFallbackMessage)

	expected := "Failed " + slackExpectedFallbackText

	if actual != expected {
		t.Errorf("Invalid failure fallback message for Slack. Expected %v, got %v", expected, actual)
	}
}
