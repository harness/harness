package server

import (
	"encoding/base32"
	"net/http"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/securecookie"

	"github.com/drone/drone/cache"
	"github.com/drone/drone/model"
	"github.com/drone/drone/router/middleware/session"
	"github.com/drone/drone/shared/token"
	"github.com/drone/drone/store"
)

func GetSelf(c *gin.Context) {
	c.JSON(200, session.User(c))
}

func GetFeed(c *gin.Context) {
	latest, _ := strconv.ParseBool(c.Query("latest"))

	repos, err := cache.GetRepos(c, session.User(c))
	if err != nil {
		c.String(500, "Error fetching repository list. %s", err)
		return
	}

	feed, err := store.GetUserFeed(c, repos, latest)
	if err != nil {
		c.String(500, "Error fetching feed. %s", err)
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

	if flush {
		log.Debugf("Evicting repository cache for user %s.", user.Login)
		cache.DeleteRepos(c, user)
	}

	remote, err := cache.GetRepos(c, user)
	if err != nil {
		c.String(500, "Error fetching repository list. %s", err)
		return
	}

	repos, err := store.GetRepoListOf(c, remote)
	if err != nil {
		c.String(500, "Error fetching repository list. %s", err)
		return
	}

	if !all {
		c.JSON(http.StatusOK, repos)
		return
	}

	// below we combine the two lists to include both active and inactive
	// repositories. This is displayed on the settings screen to enable
	// toggling on / off repository settings.

	repom := map[string]bool{}
	for _, repo := range repos {
		repom[repo.FullName] = true
	}

	for _, repo := range remote {
		if repom[repo.FullName] {
			continue
		}
		repos = append(repos, &model.Repo{
			Avatar:   repo.Avatar,
			FullName: repo.FullName,
			Owner:    repo.Owner,
			Name:     repo.Name,
		})
	}
	c.JSON(http.StatusOK, repos)
}

func GetRemoteRepos(c *gin.Context) {
	repos, err := cache.GetRepos(c, session.User(c))
	if err != nil {
		c.String(500, "Error fetching repository list. %s", err)
		return
	}
	c.JSON(http.StatusOK, repos)
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
