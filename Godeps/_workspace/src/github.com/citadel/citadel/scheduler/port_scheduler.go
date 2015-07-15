package scheduler

import "github.com/drone/drone/Godeps/_workspace/src/github.com/citadel/citadel"

// PortScheduler will refuse to schedule a container where bound Ports conflict with other instances
type PortScheduler struct{}

func (s *PortScheduler) Schedule(i *citadel.Image, e *citadel.Engine) (bool, error) {
	containers, err := e.ListContainers(false, false, "")
	if err != nil {
		return false, err
	}

	if s.hasConflictingPorts(i.BindPorts, containers) {
		return false, nil
	}

	return true, nil
}

func (s *PortScheduler) hasConflictingPorts(imagePorts []*citadel.Port, containers []*citadel.Container) bool {
	for _, ct := range containers {
		for _, hostPort := range ct.Ports {
			for _, imagePort := range imagePorts {
				// an Image with BindPorts containing HostIp == "" is equivalent to "0.0.0.0"
				// as dockerclient does this translation when running an image
				if (hostPort.HostIp == imagePort.HostIp || hostPort.HostIp == "0.0.0.0" && imagePort.HostIp == "") && hostPort.Port == imagePort.Port {
					return true
				}
			}
		}
	}

	return false
}
