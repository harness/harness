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
	"net/http"

	"github.com/drone/drone/model"
	"github.com/drone/drone/shared/token"
	"github.com/drone/drone/store"

	"github.com/gin-gonic/gin"
)

func User(c *gin.Context) *model.User {
	v, ok := c.Get("user")
	if !ok {
		return nil
	}
	u, ok := v.(*model.User)
	if !ok {
		return nil
	}
	return u
}

func Token(c *gin.Context) *token.Token {
	v, ok := c.Get("token")
	if !ok {
		return nil
	}
	u, ok := v.(*token.Token)
	if !ok {
		return nil
	}
	return u
}

func SetUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		var user *model.User

		t, err := token.ParseRequest(c.Request, func(t *token.Token) (string, error) {
			var err error
			user, err = store.GetUserLogin(c, t.Text)
			return user.Hash, err
		})
		if err == nil {
			confv := c.MustGet("config")
			if conf, ok := confv.(*model.Settings); ok {
				user.Admin = conf.IsAdmin(user)
			}
			c.Set("user", user)

			// if this is a session token (ie not the API token)
			// this means the user is accessing with a web browser,
			// so we should implement CSRF protection measures.
			if t.Kind == token.SessToken {
				err = token.CheckCsrf(c.Request, func(t *token.Token) (string, error) {
					return user.Hash, nil
				})
				// if csrf token validation fails, exit immediately
				// with a not authorized error.
				if err != nil {
					c.AbortWithStatus(http.StatusUnauthorized)
					return
				}
			}
		}
		c.Next()
	}
}

func MustAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := User(c)
		switch {
		case user == nil:
			c.String(401, "User not authorized")
			c.Abort()
		case user.Admin == false:
			c.String(413, "User not authorized")
			c.Abort()
		default:
			c.Next()
		}
	}
}

func MustRepoAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := User(c)
		perm := Perm(c)
		switch {
		case user == nil:
			c.String(401, "User not authorized")
			c.Abort()
		case perm.Admin == false:
			c.String(403, "User not authorized")
			c.Abort()
		default:
			c.Next()
		}
	}
}

func MustUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := User(c)
		switch {
		case user == nil:
			c.String(401, "User not authorized")
			c.Abort()
		default:
			c.Next()
		}
	}
}
