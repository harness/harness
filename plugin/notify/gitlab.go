package notify

import (
	"fmt"
	"strconv"

	"github.com/Bugagazavr/go-gitlab-client"
	"github.com/drone/drone/plugin/remote/gitlab"
	"github.com/drone/drone/shared/model"
)

type Gitlab struct {
	SkipVerify bool `yaml:"skip_verify,omitempty"`
	Started    bool `yaml:"on_started,omitempty"`
	Success    bool `yaml:"on_success,omitempty"`
	Failure    bool `yaml:"on_failure,omitempty"`
}

type GitlabClient interface {
	SendRepoCommitComment(id string, sha string, body string) (*gogitlab.CommitComment, error)
}

const (
	StatusPending = ":raised_hand:"
	StatusSuccess = ":thumbsup:"
	StatusFailure = ":thumbsdown:"
	StatusError   = ":exclamation:"
)

const (
	DescPending = "this build is pending"
	DescSuccess = "the build was successful"
	DescFailure = "the build failed"
	DescError   = "oops, something went wrong"
)

const (
	PRMasterBranch = "master"
	PRBadToMerge   = " -> bad to merge"
	PRGoodToMerge  = " -> good to merge"
)

// Send uses the Gitlab repository API to comment the commit
func (g *Gitlab) Send(context *model.Request) error {
	if !g.isRequested(context) {
		return nil
	}

	return g.send(
		context,
		gitlab.NewClient(fmt.Sprintf("http://%s", context.Repo.Host), context.User.Access, g.SkipVerify),
	)
}

func (g *Gitlab) isRequested(context *model.Request) bool {
	if context.Repo.Remote != model.RemoteGitlab {
		return false
	}

	switch context.Commit.Status {
	case model.StatusStarted:
		if !g.Started {
			return false
		}
	case model.StatusSuccess:
		if !g.Success {
			return false
		}
	case model.StatusFailure, model.StatusError, model.StatusKilled:
		if !g.Failure {
			return false
		}
	}

	return true
}

func (g *Gitlab) send(context *model.Request, client GitlabClient) error {
	msg := fmt.Sprintf(
		"[%s](%s) %s%s",
		getDesc(context.Commit.Status),
		getBuildUrl(context),
		getStatus(context.Commit.Status),
		getMergeRequestComment(context.Commit.Branch, context.Commit.Status),
	)

	_, err := client.SendRepoCommitComment(strconv.FormatInt(context.Repo.ID, 10), context.Commit.Sha, msg)

	return err
}

// getStatus converts a Drone status to a Gitlab status.
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

// getDesc generates a description message for the comment based on the status.
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

func getMergeRequestComment(branch, status string) string {
	if branch != PRMasterBranch {
		switch status {
		case model.StatusSuccess:
			return PRGoodToMerge
		case model.StatusFailure:
			return PRBadToMerge
		}
	}
	return ""
}
