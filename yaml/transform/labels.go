package transform

import (
	"github.com/drone/drone/yaml"
)

// Labels transforms the steps in the Yaml pipeline to include a Labels if it doens't exist
func Labels(c *yaml.Config) error {
	var images []*yaml.Container
	images = append(images, c.Pipeline...)
	images = append(images, c.Services...)

	for _, p := range images {
		if p.Labels == nil {
			p.Labels = map[string]string{}
		}
	}
	return nil
}
