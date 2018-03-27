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

package server

import (
	"github.com/drone/drone/store"
	"github.com/drone/drone/version"
	"github.com/gin-gonic/gin"
)

// Health endpoint returns a 500 if the server state is unhealthy.
func Health(c *gin.Context) {
	if err := store.FromContext(c).Ping(); err != nil {
		c.String(500, err.Error())
		return
	}
	c.String(200, "")
}

// Version endpoint returns the server version and build information.
func Version(c *gin.Context) {
	c.JSON(200, gin.H{
		"source":  "https://github.com/drone/drone",
		"version": version.Version.String(),
	})
}
