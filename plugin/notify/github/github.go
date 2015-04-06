package github

import (
	"fmt"
	"net/url"

	"code.google.com/p/goauth2/oauth"
	"github.com/drone/drone/shared/model"
	"github.com/google/go-github/github"
)

const (
	NotifyDisabled = "disabled"
	NotifyFalse    = "false"
	NotifyOff      = "off"
)

const (
	StatusPending = "pending"
	StatusSuccess = "success"
	StatusFailure = "failure"
	StatusError   = "error"
)

const (
	DescPending = "this build is pending"
	DescSuccess = "the build was successful"
	DescFailure = "the build failed"
	DescError   = "oops, something went wrong"
)

const (
	BaseURL = "https://api.github.com/"
)

type GitHub string

// Send uses the github status API to update the build
// status in github or github enterprise.
func (g GitHub) Send(context *model.Request) error {

	// a user can toggle the status api on / off
	// in the .drone.yml
	switch g {
	case NotifyDisabled, NotifyOff, NotifyFalse:
		return nil
	}

	// this should only be executed for GitHub and
	// GitHub enterprise requests.
	switch context.Repo.Remote {
	case model.RemoteGithub, model.RemoteGithubEnterprise:
		break
	default:
		return nil
	}

	var target = getTarget(
		context.Host,
		context.Repo.Host,
		context.Repo.Owner,
		context.Repo.Name,
		context.Commit.Branch,
		context.Commit.Sha,
	)

	return send(
		context.Repo.URL,
		context.Repo.Host,
		context.Repo.Owner,
		context.Repo.Name,
		getStatus(context.Commit.Status),
		getDesc(context.Commit.Status),
		target,
		context.Commit.Sha,
		context.User.Access,
	)
}

func send(rawurl, host, owner, repo, status, desc, target, ref, token string) error {
	transport := &oauth.Transport{
		Token: &oauth.Token{AccessToken: token},
	}

	data := github.RepoStatus{
		Context:     github.String("Drone"),
		State:       github.String(status),
		Description: github.String(desc),
		TargetURL:   github.String(target),
	}

	client := github.NewClient(transport.Client())

	// if this is for github enterprise we need to set
	// the base url. Per the documentation, we need to
	// ensure there is a trailing slash.
	if host != model.RemoteGithub {
		client.BaseURL, _ = getEndpoint(rawurl)
	}

	_, _, err := client.Repositories.CreateStatus(owner, repo, ref, &data)
	return err
}

// getStatus is a helper functin that converts a Drone
// status to a GitHub status.
func getStatus(status string) string {
	switch status {
	case model.StatusEnqueue, model.StatusStarted:
		return StatusPending
	case model.StatusSuccess:
		return StatusSuccess
	case model.StatusFailure:
		return StatusFailure
	case model.StatusError, model.StatusKilled:
		return StatusError
	default:
		return StatusError
	}
}

// getDesc is a helper function that generates a description
// message for the build based on the status.
func getDesc(status string) string {
	switch status {
	case model.StatusEnqueue, model.StatusStarted:
		return DescPending
	case model.StatusSuccess:
		return DescSuccess
	case model.StatusFailure:
		return DescFailure
	case model.StatusError, model.StatusKilled:
		return DescError
	default:
		return DescError
	}
}

// getTarget is a helper function that generates a URL
// for the user to click and jump to the build results.
//
// for example:
//   https://drone.io/github.com/drone/drone-test-go/master/c22aec9c53
func getTarget(url, host, owner, repo, branch, commit string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s/%s", url, host, owner, repo, branch, commit)
}

// getEndpoint is a helper funcation that parsed the
// repository HTML URL to determine the API URL. It is
// intended for use with GitHub enterprise.
func getEndpoint(rawurl string) (*url.URL, error) {
	uri, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	uri.Path = "/api/v3/"
	return uri, nil
}
