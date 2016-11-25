package transform

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/drone/drone/yaml"
)

// ImagePull transforms the Yaml to automatically pull the latest image.
func ImagePull(conf *yaml.Config, pull bool) error {
	for _, plugin := range conf.Pipeline {
		if !isPlugin(plugin) {
			continue
		}
		plugin.Pull = pull
	}
	return nil
}

// ImageTag transforms the Yaml to use the :latest image tag when empty.
func ImageTag(conf *yaml.Config) error {
	for _, image := range conf.Pipeline {
		if !strings.Contains(image.Image, ":") {
			image.Image = image.Image + ":latest"
		}
	}
	for _, image := range conf.Services {
		if !strings.Contains(image.Image, ":") {
			image.Image = image.Image + ":latest"
		}
	}
	return nil
}

// ImageEscalate transforms the Yaml to automatically enable privileged mode
// for a subset of white-listed plugins matching the given patterns.
func ImageEscalate(conf *yaml.Config, patterns []string) error {
	for _, c := range conf.Pipeline {
		for _, pattern := range patterns {
			if ok, _ := filepath.Match(pattern, c.Image); ok {
				if c.Detached {
					return fmt.Errorf("Detached mode disabled for the %s plugin", c.Image)
				}
				if len(c.Entrypoint) != 0 {
					return fmt.Errorf("Custom entrypoint disabled for the %s plugin", c.Image)
				}
				if len(c.Command) != 0 {
					return fmt.Errorf("Custom command disabled for the %s plugin", c.Image)
				}
				if len(c.Commands) != 0 {
					return fmt.Errorf("Custom commands disabled for the %s plugin", c.Image)
				}
				c.Privileged = true
			}
		}
	}
	return nil
}
