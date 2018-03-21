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
	"github.com/drone/drone/version"
	"github.com/gin-gonic/gin"
)

// Version is a middleware function that appends the Drone version information
// to the HTTP response. This is intended for debugging and troubleshooting.
func Version(c *gin.Context) {
	c.Header("X-DRONE-VERSION", version.Version.String())
}
