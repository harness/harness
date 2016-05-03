package github

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/drone/drone/model"
)

const (
	hookEvent  = "X-Github-Event"
	hookField  = "payload"
	hookDeploy = "deployment"
	hookPush   = "push"
	hookPull   = "pull_request"

	actionOpen = "opened"
	actionSync = "synchronize"

	stateOpen = "open"
)

// parseHook parses a Bitbucket hook from an http.Request request and returns
// Repo and Build detail. If a hook type is unsupported nil values are returned.
func parseHook(r *http.Request, merge bool) (*model.Repo, *model.Build, error) {
	var reader io.Reader = r.Body

	if payload := r.FormValue(hookField); payload != "" {
		reader = bytes.NewBufferString(payload)
	}

	raw, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, nil, err
	}

	switch r.Header.Get(hookEvent) {
	case hookPush:
		return parsePushHook(raw)
	case hookDeploy:
		return parseDeployHook(raw)
	case hookPull:
		return parsePullHook(raw, merge)
	}
	return nil, nil, nil
}

// parsePushHook parses a push hook and returns the Repo and Build details.
// If the commit type is unsupported nil values are returned.
func parsePushHook(payload []byte) (*model.Repo, *model.Build, error) {
	hook := new(webhook)
	err := json.Unmarshal(payload, hook)
	if err != nil {
		return nil, nil, err
	}
	if hook.Deleted {
		return nil, nil, err
	}
	return convertRepoHook(hook), convertPushHook(hook), nil
}

// parseDeployHook parses a deployment and returns the Repo and Build details.
// If the commit type is unsupported nil values are returned.
func parseDeployHook(payload []byte) (*model.Repo, *model.Build, error) {
	hook := new(webhook)
	if err := json.Unmarshal(payload, hook); err != nil {
		return nil, nil, err
	}
	return convertRepoHook(hook), convertDeployHook(hook), nil
}

// parsePullHook parses a pull request hook and returns the Repo and Build
// details. If the pull request is closed nil values are returned.
func parsePullHook(payload []byte, merge bool) (*model.Repo, *model.Build, error) {
	hook := new(webhook)
	err := json.Unmarshal(payload, hook)
	if err != nil {
		return nil, nil, err
	}

	// ignore these
	if hook.Action != actionOpen && hook.Action != actionSync {
		return nil, nil, nil
	}
	if hook.PullRequest.State != stateOpen {
		return nil, nil, nil
	}
	return convertRepoHook(hook), convertPullHook(hook, merge), nil
}
