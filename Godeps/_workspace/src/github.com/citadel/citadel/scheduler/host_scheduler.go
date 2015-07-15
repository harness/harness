package scheduler

import (
	"strings"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/citadel/citadel"
)

type HostScheduler struct {
}

func (h *HostScheduler) Schedule(c *citadel.Image, e *citadel.Engine) (bool, error) {
	if len(c.Labels) == 0 {
		return true, nil
	}

	return h.validHost(e, c.Labels), nil
}

func (h *HostScheduler) validHost(e *citadel.Engine, labels []string) bool {
	for _, label := range labels {
		parts := strings.Split(label, "host:")
		if len(parts) != 2 {
			return false
		}

		host := parts[1]
		if e.ID == host {
			return true
		}
	}

	return false
}
