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
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/drone/drone/router/middleware/session"
	"github.com/drone/drone/store"
	"github.com/gin-gonic/gin"
)

// FileList gets a list file by build.
func FileList(c *gin.Context) {
	num, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	repo := session.Repo(c)
	build, err := store.FromContext(c).GetBuildNumber(repo, num)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	files, err := store.FromContext(c).FileList(build)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(200, files)
}

// FileGet gets a file by process and name
func FileGet(c *gin.Context) {
	var (
		repo = session.Repo(c)
		name = strings.TrimPrefix(c.Param("file"), "/")
		raw  = func() bool {
			return c.DefaultQuery("raw", "false") == "true"
		}()
	)

	num, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	pid, err := strconv.Atoi(c.Param("proc"))
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	build, err := store.FromContext(c).GetBuildNumber(repo, num)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	proc, err := store.FromContext(c).ProcFind(build, pid)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	file, err := store.FromContext(c).FileFind(proc, name)
	if err != nil {
		c.String(404, "Error getting file %q. %s", name, err)
		return
	}

	if !raw {
		c.JSON(200, file)
		return
	}

	rc, err := store.FromContext(c).FileRead(proc, file.Name)
	if err != nil {
		c.String(404, "Error getting file stream %q. %s", name, err)
		return
	}
	defer rc.Close()

	switch file.Mime {
	case "application/vnd.drone.test+json":
		c.Header("Content-Type", "application/json")
	}

	io.Copy(c.Writer, rc)
}
