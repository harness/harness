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

package session

import (
	"github.com/drone/drone/shared/token"
	"github.com/gin-gonic/gin"
)

// AuthorizeAgent authorizes requsts from build agents to access the queue.
func AuthorizeAgent(c *gin.Context) {
	secret := c.MustGet("agent").(string)
	if secret == "" {
		c.String(401, "invalid or empty token.")
		return
	}

	parsed, err := token.ParseRequest(c.Request, func(t *token.Token) (string, error) {
		return secret, nil
	})
	if err != nil {
		c.String(500, "invalid or empty token. %s", err)
		c.Abort()
	} else if parsed.Kind != token.AgentToken {
		c.String(403, "invalid token. please use an agent token")
		c.Abort()
	} else {
		c.Next()
	}
}
