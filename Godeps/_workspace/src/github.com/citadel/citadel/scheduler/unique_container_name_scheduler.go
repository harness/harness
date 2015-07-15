package scheduler

import (
	"github.com/drone/drone/Godeps/_workspace/src/github.com/citadel/citadel"
)

// UniqueContainerNameScheduler only returns engines that do not already have a container with the same name running
type UniqueContainerNameScheduler struct {
}

func (u *UniqueContainerNameScheduler) Schedule(c *citadel.Image, e *citadel.Engine) (bool, error) {
	containers, err := e.ListContainers(false, false, "")
	if err != nil {
		return false, err
	}

	if u.containerNameExists(c, containers) {
		return false, nil
	}

	return true, nil
}

func (u *UniqueContainerNameScheduler) containerNameExists(i *citadel.Image, containers []*citadel.Container) bool {
	if i.ContainerName == "" {
		return false
	}

	for _, c := range containers {
		if i.ContainerName == c.Name {
			return true
		}
	}

	return false
}
