package transform

import (
	"os"

	"github.com/drone/drone/yaml"
)

var (
	httpProxy  = os.Getenv("HTTP_PROXY")
	httpsProxy = os.Getenv("HTTPS_PROXY")
	noProxy    = os.Getenv("NO_PROXY")
)

// Environ transforms the steps in the Yaml pipeline to include runtime
// environment variables.
func Environ(c *yaml.Config, envs map[string]string) error {
	var images []*yaml.Container
	images = append(images, c.Pipeline...)
	images = append(images, c.Services...)

	for _, p := range images {
		if p.Environment == nil {
			p.Environment = map[string]string{}
		}
		for k, v := range envs {
			if v == "" {
				continue
			}
			p.Environment[k] = v
		}
		if httpProxy != "" {
			p.Environment["HTTP_PROXY"] = httpProxy
			p.Environment["http_proxy"] = httpProxy
		}
		if httpsProxy != "" {
			p.Environment["HTTPS_PROXY"] = httpsProxy
			p.Environment["https_proxy"] = httpsProxy
		}
		if noProxy != "" {
			p.Environment["NO_PROXY"] = noProxy
			p.Environment["no_proxy"] = noProxy
		}
	}
	return nil
}
