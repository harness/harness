package transform

import "github.com/drone/drone/yaml"

const clone = "clone"

// Clone transforms the Yaml to include a clone step.
func Clone(c *yaml.Config, plugin string) error {
	if plugin == "" {
		plugin = "git"
	}

	for _, p := range c.Pipeline {
		if p.Name == clone {
			if p.Image == "" {
				p.Image = plugin
			}
			return nil
		}
	}

	s := &yaml.Container{
		Image: plugin,
		Name:  clone,
	}

	c.Pipeline = append([]*yaml.Container{s}, c.Pipeline...)
	return nil
}
