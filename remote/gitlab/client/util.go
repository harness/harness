package client

import (
	"net/url"
	"strings"
)

func encodeParameter(value string) string {
	return strings.Replace(url.QueryEscape(value), "/", "%2F", 0)
}

// Tag returns current tag for push event hook payload
// This function returns empty string for any other events
func (h *HookPayload) Tag() string {
	return strings.TrimPrefix(h.Ref, "refs/tags/")
}

// Branch returns current branch for push event hook payload
// This function returns empty string for any other events
func (h *HookPayload) Branch() string {
	return strings.TrimPrefix(h.Ref, "refs/heads/")
}

// Head returns the latest changeset for push event hook payload
func (h *HookPayload) Head() hCommit {
	c := hCommit{}
	for _, cm := range h.Commits {
		if h.After == cm.Id {
			return cm
		}
	}
	return c
}
