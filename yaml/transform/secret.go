package transform

import (
	"github.com/drone/drone/model"
	"github.com/drone/drone/yaml"
)

//
// TODO remove
//

func ImageSecrets(c *yaml.Config, secrets []*model.Secret, event string) error {
	var images []*yaml.Container
	images = append(images, c.Pipeline...)
	images = append(images, c.Services...)

	for _, image := range images {
		imageSecrets(image, secrets, event)
	}
	return nil
}

func imageSecrets(c *yaml.Container, secrets []*model.Secret, event string) {
	for _, secret := range secrets {
		if !secret.Match(c.Image, event) {
			continue
		}

		switch secret.Name {
		case "REGISTRY_USERNAME":
			c.AuthConfig.Username = secret.Value
		case "REGISTRY_PASSWORD":
			c.AuthConfig.Password = secret.Value
		case "REGISTRY_EMAIL":
			c.AuthConfig.Email = secret.Value
		default:
			if c.Environment == nil {
				c.Environment = map[string]string{}
			}
			c.Environment[secret.Name] = secret.Value
		}
	}
}
