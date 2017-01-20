package transform

import "github.com/drone/drone/yaml"

// ImageVolume mounts a default volume (used for drone exec)
func ImageVolume(conf *yaml.Config, volumes []string) error {

	if len(volumes) == 0 {
		return nil
	}

	var containers []*yaml.Container
	containers = append(containers, conf.Pipeline...)
	containers = append(containers, conf.Services...)

	for _, container := range containers {
		container.Volumes = append(container.Volumes, volumes...)
	}

	return nil
}
