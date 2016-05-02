package bitbucket

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote/bitbucket/internal"
)

const (
	hookEvent       = "X-Event-Key"
	hookPush        = "repo:push"
	hookPullCreated = "pullrequest:created"
	hookPullUpdated = "pullrequest:updated"

	changeBranch      = "branch"
	changeNamedBranch = "named_branch"

	stateMerged   = "MERGED"
	stateDeclined = "DECLINED"
	stateOpen     = "OPEN"
)

// parseHook parses a Bitbucket hook from an http.Request request and returns
// Repo and Build detail. If a hook type is unsupported nil values are returned.
func parseHook(r *http.Request) (*model.Repo, *model.Build, error) {
	payload, _ := ioutil.ReadAll(r.Body)

	switch r.Header.Get(hookEvent) {
	case hookPush:
		return parsePushHook(payload)
	case hookPullCreated, hookPullUpdated:
		return parsePullHook(payload)
	}
	return nil, nil, nil
}

// parsePushHook parses a push hook and returns the Repo and Build details.
// If the commit type is unsupported nil values are returned.
func parsePushHook(payload []byte) (*model.Repo, *model.Build, error) {
	hook := internal.PushHook{}

	err := json.Unmarshal(payload, &hook)
	if err != nil {
		return nil, nil, err
	}

	for _, change := range hook.Push.Changes {
		if change.New.Target.Hash == "" {
			continue
		}
		return convertRepo(&hook.Repo), convertPushHook(&hook, &change), nil
	}
	return nil, nil, nil
}

// parsePullHook parses a pull request hook and returns the Repo and Build
// details. If the pull request is closed nil values are returned.
func parsePullHook(payload []byte) (*model.Repo, *model.Build, error) {
	hook := internal.PullRequestHook{}

	if err := json.Unmarshal(payload, &hook); err != nil {
		return nil, nil, err
	}
	if hook.PullRequest.State != stateOpen {
		return nil, nil, nil
	}
	return convertRepo(&hook.Repo), convertPullHook(&hook), nil
}
