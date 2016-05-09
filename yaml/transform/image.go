package transform

import (
	"path/filepath"
	"strings"

	"github.com/drone/drone/yaml"
)

func ImagePull(conf *yaml.Config, pull bool) error {
	for _, plugin := range conf.Pipeline {
		if len(plugin.Commands) == 0 || len(plugin.Vargs) == 0 {
			continue
		}
		plugin.Pull = pull
	}
	return nil
}

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

func ImageName(conf *yaml.Config) error {
	for _, image := range conf.Pipeline {
		image.Image = strings.Replace(image.Image, "_", "-", -1)
	}
	return nil
}

func ImageNamespace(conf *yaml.Config, namespace string) error {
	for _, image := range conf.Pipeline {
		if strings.Contains(image.Image, "/") {
			continue
		}
		if len(image.Vargs) == 0 {
			continue
		}
		image.Image = filepath.Join(namespace, image.Image)
	}
	return nil
}

func ImageEscalate(conf *yaml.Config, patterns []string) error {
	for _, c := range conf.Pipeline {
		for _, pattern := range patterns {
			if ok, _ := filepath.Match(pattern, c.Image); ok {
				c.Privileged = true
			}
		}
	}
	return nil
}
