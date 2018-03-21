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
	"encoding/base32"
	"net/http"
	"strconv"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/securecookie"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote"
	"github.com/drone/drone/router/middleware/session"
	"github.com/drone/drone/shared/token"
	"github.com/drone/drone/store"
)

func GetSelf(c *gin.Context) {
	c.JSON(200, session.User(c))
}

func GetFeed(c *gin.Context) {
	user := session.User(c)
	latest, _ := strconv.ParseBool(c.Query("latest"))

	if time.Unix(user.Synced, 0).Add(time.Hour * 72).Before(time.Now()) {
		logrus.Debugf("sync begin: %s", user.Login)

		user.Synced = time.Now().Unix()
		store.FromContext(c).UpdateUser(user)

		sync := syncer{
			remote:  remote.FromContext(c),
			store:   store.FromContext(c),
			perms:   store.FromContext(c),
			limiter: Config.Services.Limiter,
		}
		if err := sync.Sync(user); err != nil {
			logrus.Debugf("sync error: %s: %s", user.Login, err)
		} else {
			logrus.Debugf("sync complete: %s", user.Login)
		}
	}

	if latest {
		feed, err := store.FromContext(c).RepoListLatest(user)
		if err != nil {
			c.String(500, "Error fetching feed. %s", err)
		} else {
			c.JSON(200, feed)
		}
		return
	}

	feed, err := store.FromContext(c).UserFeed(user)
	if err != nil {
		c.String(500, "Error fetching user feed. %s", err)
		return
	}
	c.JSON(200, feed)
}

func GetRepos(c *gin.Context) {
	var (
		user     = session.User(c)
		all, _   = strconv.ParseBool(c.Query("all"))
		flush, _ = strconv.ParseBool(c.Query("flush"))
	)

	if flush || time.Unix(user.Synced, 0).Add(time.Hour*72).Before(time.Now()) {
		logrus.Debugf("sync begin: %s", user.Login)
		user.Synced = time.Now().Unix()
		store.FromContext(c).UpdateUser(user)

		sync := syncer{
			remote:  remote.FromContext(c),
			store:   store.FromContext(c),
			perms:   store.FromContext(c),
			limiter: Config.Services.Limiter,
		}
		if err := sync.Sync(user); err != nil {
			logrus.Debugf("sync error: %s: %s", user.Login, err)
		} else {
			logrus.Debugf("sync complete: %s", user.Login)
		}
	}

	repos, err := store.FromContext(c).RepoList(user)
	if err != nil {
		c.String(500, "Error fetching repository list. %s", err)
		return
	}

	if all {
		c.JSON(http.StatusOK, repos)
		return
	}

	active := []*model.Repo{}
	for _, repo := range repos {
		if repo.IsActive {
			active = append(active, repo)
		}
	}
	c.JSON(http.StatusOK, active)
}

func PostToken(c *gin.Context) {
	user := session.User(c)

	token := token.New(token.UserToken, user.Login)
	tokenstr, err := token.Sign(user.Hash)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.String(http.StatusOK, tokenstr)
}

func DeleteToken(c *gin.Context) {
	user := session.User(c)
	user.Hash = base32.StdEncoding.EncodeToString(
		securecookie.GenerateRandomKey(32),
	)
	if err := store.UpdateUser(c, user); err != nil {
		c.String(500, "Error revoking tokens. %s", err)
		return
	}

	token := token.New(token.UserToken, user.Login)
	tokenstr, err := token.Sign(user.Hash)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.String(http.StatusOK, tokenstr)
}
