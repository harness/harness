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

// ImageName transforms the Yaml to replace underscores with dashes.
func ImageName(conf *yaml.Config) error {
	for _, image := range conf.Pipeline {
		image.Image = strings.Replace(image.Image, "_", "-", -1)
	}
	return nil
}

// ImageNamespace transforms the Yaml to use a default namepsace for plugins.
func ImageNamespace(conf *yaml.Config, namespace string) error {
	for _, image := range conf.Pipeline {
		if strings.Contains(image.Image, "/") {
			continue
		}
		if !isPlugin(image) {
			continue
		}
		image.Image = filepath.Join(namespace, image.Image)
	}
	return nil
}

// ImageEscalate transforms the Yaml to automatically enable privileged mode
// for a subset of white-listed plugins matching the given patterns.
func ImageEscalate(conf *yaml.Config, patterns []string) error {
	for _, c := range conf.Pipeline {
		for _, pattern := range patterns {
			if ok, _ := filepath.Match(pattern, c.Image); ok {
				if len(c.Commands) != 0 {
					return fmt.Errorf("Custom commands disabled for the %s plugin", c.Image)
				}
				c.Privileged = true
			}
		}
	}
	return nil
}
