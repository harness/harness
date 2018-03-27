// Copyright 2018 Drone.IO Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
