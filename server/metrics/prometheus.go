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

package metrics

import (
	"errors"
	"fmt"

	"github.com/drone/drone/server"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// errInvalidToken is returned when the api request token is invalid.
	errInvalidToken = errors.New("Invalid or missing token")
)

// PromHandler will pass the call from /api/metrics/prometheus to prometheus
func PromHandler() gin.HandlerFunc {
	handler := promhttp.Handler()

	return func(c *gin.Context) {
		token := server.Config.Prometheus.AuthToken

		if token == "" {
			handler.ServeHTTP(c.Writer, c.Request)
			return
		}

		header := c.Request.Header.Get("Authorization")

		if header == "" {
			c.String(401, errInvalidToken.Error())
			return
		}

		bearer := fmt.Sprintf("Bearer %s", token)

		if header != bearer {
			c.String(401, errInvalidToken.Error())
			return
		}

		handler.ServeHTTP(c.Writer, c.Request)
	}
}
