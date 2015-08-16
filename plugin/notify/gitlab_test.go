package notify

import (
	"fmt"
	"testing"

	"github.com/Bugagazavr/go-gitlab-client"
	"github.com/drone/drone/shared/model"
)

type MockGitlabClient struct {
	Comment *gogitlab.CommitComment
	id      string
	sha     string
	body    string
}

func (c *MockGitlabClient) SendRepoCommitComment(id string, sha string, body string) (*gogitlab.CommitComment, error) {
	c.id = id
	c.sha = sha
	c.body = body
	return c.Comment, nil
}

var gitlabClient = &MockGitlabClient{}

var gitlabSubject = &Gitlab{
	SkipVerify: false,
	Started:    true,
	Success:    true,
	Failure:    true,
}

var gitlabRequest = &model.Request{
	Host: "http://examplehost.com",
	Repo: &model.Repo{
		ID:     123456,
		Host:   "examplegit.com",
		Owner:  "owner",
		Name:   "repo",
		Remote: model.RemoteGitlab,
	},
	Commit: &model.Commit{
		Sha:    "abc",
		Branch: "example",
	},
	User: &model.User{
		Access: "secret_token",
	},
}

func Test_GitlabSendStarted(t *testing.T) {
	gitlabRequest.Commit.Status = "Started"

	expected := gogitlab.CommitComment{
		Note: fmt.Sprintf("[this build is pending](%s) :raised_hand:", getBuildUrl(gitlabRequest)),
	}

	gitlabClient.Comment = &expected

	err := gitlabSubject.send(gitlabRequest, gitlabClient)
	if err != nil {
		t.Errorf("Unexepected error: %s", err.Error())
	}

	if *gitlabClient.Comment != expected {
		t.Errorf("Invalid gitlab payload. Expected: %v, got %v", expected, *gitlabClient.Comment)
	}
}

func Test_GitlabSendSuccess(t *testing.T) {
	gitlabRequest.Commit.Status = "Success"

	expected := gogitlab.CommitComment{
		Note: fmt.Sprintf("[the build was successful](%s) :thumbsup: -> good to merge", getBuildUrl(gitlabRequest)),
	}

	gitlabClient.Comment = &expected

	err := gitlabSubject.send(gitlabRequest, gitlabClient)
	if err != nil {
		t.Errorf("Unexepected error: %s", err.Error())
	}

	if *gitlabClient.Comment != expected {
		t.Errorf("Invalid gitlab payload. Expected: %v, got %v", expected, *gitlabClient.Comment)
	}
}

func Test_GitlabSendFailure(t *testing.T) {
	gitlabRequest.Commit.Status = "Failure"

	expected := gogitlab.CommitComment{
		Note: fmt.Sprintf("[the build failed](%s) :thumbsdown: -> bad to merge", getBuildUrl(gitlabRequest)),
	}

	gitlabClient.Comment = &expected

	err := gitlabSubject.send(gitlabRequest, gitlabClient)
	if err != nil {
		t.Errorf("Unexepected error: %s", err.Error())
	}

	if *gitlabClient.Comment != expected {
		t.Errorf("Invalid gitlab payload. Expected: %v, got %v", expected, *gitlabClient.Comment)
	}
}
