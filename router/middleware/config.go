package middleware

import (
	"github.com/drone/drone/model"

	"github.com/gin-gonic/gin"
	"github.com/urfave/cli"
)

const configKey = "config"

// Config is a middleware function that initializes the Configuration and
// attaches to the context of every http.Request.
func Config(cli *cli.Context) gin.HandlerFunc {
	v := setupConfig(cli)
	return func(c *gin.Context) {
		c.Set(configKey, v)
	}
}

// helper function to create the configuration from the CLI context.
func setupConfig(c *cli.Context) *model.Settings {
	return &model.Settings{
		Open:   c.Bool("open"),
		Secret: c.String("agent-secret"),
		Admins: sliceToMap2(c.StringSlice("admin")),
		Orgs:   sliceToMap2(c.StringSlice("orgs")),
	}
}

// helper function to convert a string slice to a map.
func sliceToMap2(s []string) map[string]bool {
	v := map[string]bool{}
	for _, ss := range s {
		if ss == "" {
			continue
		}
		v[ss] = true
	}
	return v
}
