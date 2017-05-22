package session

import (
	"net/http"

	"github.com/drone/drone/cache"
	"github.com/drone/drone/model"
	"github.com/drone/drone/store"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
)

func Repo(c *gin.Context) *model.Repo {
	v, ok := c.Get("repo")
	if !ok {
		return nil
	}
	r, ok := v.(*model.Repo)
	if !ok {
		return nil
	}
	return r
}

func Repos(c *gin.Context) []*model.RepoLite {
	v, ok := c.Get("repos")
	if !ok {
		return nil
	}
	r, ok := v.([]*model.RepoLite)
	if !ok {
		return nil
	}
	return r
}

func SetRepo() gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			owner = c.Param("owner")
			name  = c.Param("name")
			user  = User(c)
		)

		repo, err := store.GetRepoOwnerName(c, owner, name)
		if err == nil {
			c.Set("repo", repo)
			c.Next()
			return
		}

		// debugging
		log.Debugf("Cannot find repository %s/%s. %s",
			owner,
			name,
			err.Error(),
		)

		if user != nil {
			c.AbortWithStatus(http.StatusNotFound)
		} else {
			c.AbortWithStatus(http.StatusUnauthorized)
		}
	}
}

func Perm(c *gin.Context) *model.Perm {
	v, ok := c.Get("perm")
	if !ok {
		return nil
	}
	u, ok := v.(*model.Perm)
	if !ok {
		return nil
	}
	return u
}

func SetPerm() gin.HandlerFunc {

	return func(c *gin.Context) {
		user := User(c)
		repo := Repo(c)
		perm := &model.Perm{}

		switch {
		case user != nil && user.Admin:
			perm.Pull = true
			perm.Push = true
			perm.Admin = true

		case user != nil:
			var err error
			perm, err = cache.GetPerms(c, user, repo.Owner, repo.Name)
			if err != nil {
				log.Errorf("Error fetching permission for %s %s",
					user.Login, repo.FullName)
			}
		}

		switch {
		case repo.Visibility == model.VisibilityPublic:
			perm.Pull = true
		case repo.Visibility == model.VisibilityInternal && user != nil:
			perm.Pull = true
		}

		if user != nil {
			log.Debugf("%s granted %+v permission to %s",
				user.Login, perm, repo.FullName)

		} else {
			log.Debugf("Guest granted %+v to %s", perm, repo.FullName)
		}

		c.Set("perm", perm)
		c.Next()
	}
}

func MustPull(c *gin.Context) {
	user := User(c)
	perm := Perm(c)

	if perm.Pull {
		c.Next()
		return
	}

	// debugging
	if user != nil {
		c.AbortWithStatus(http.StatusNotFound)
		log.Debugf("User %s denied read access to %s",
			user.Login, c.Request.URL.Path)
	} else {
		c.AbortWithStatus(http.StatusUnauthorized)
		log.Debugf("Guest denied read access to %s %s",
			c.Request.Method,
			c.Request.URL.Path,
		)
	}
}

func MustPush(c *gin.Context) {
	user := User(c)
	perm := Perm(c)

	// if the user has push access, immediately proceed
	// the middleware execution chain.
	if perm.Push {
		c.Next()
		return
	}

	// debugging
	if user != nil {
		c.AbortWithStatus(http.StatusNotFound)
		log.Debugf("User %s denied write access to %s",
			user.Login, c.Request.URL.Path)

	} else {
		c.AbortWithStatus(http.StatusUnauthorized)
		log.Debugf("Guest denied write access to %s %s",
			c.Request.Method,
			c.Request.URL.Path,
		)
	}
}
