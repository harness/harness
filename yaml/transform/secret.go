package transform

import (
	"github.com/drone/drone/model"
	"github.com/drone/drone/yaml"
)

func Secret(c *yaml.Config, event string, secrets []*model.Secret) error {

	for _, p := range c.Pipeline {
		for _, secret := range secrets {

			switch secret.Name {
			case "REGISTRY_USERNAME":
				p.AuthConfig.Username = secret.Value
			case "REGISTRY_PASSWORD":
				p.AuthConfig.Password = secret.Value
			case "REGISTRY_EMAIL":
				p.AuthConfig.Email = secret.Value
			default:
				if p.Environment == nil {
					p.Environment = map[string]string{}
				}
				p.Environment[secret.Name] = secret.Value
			}

		}
	}

	return nil
}
