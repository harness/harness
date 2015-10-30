package session

import (
	"net/http"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote"
	"github.com/drone/drone/shared/token"
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
		)

		user := User(c)
		repo, err := store.GetRepoOwnerName(c, owner, name)
		if err == nil {
			c.Set("repo", repo)
			c.Next()
			return
		}

		// if the user is not nil, check the remote system
		// to see if the repository actually exists. If yes,
		// we can prompt the user to add.
		if user != nil {
			remote := remote.FromContext(c)
			repo, err = remote.Repo(user, owner, name)
			if err != nil {
				log.Errorf("Cannot find remote repository %s/%s for user %s. %s",
					owner, name, user.Login, err)
			} else {
				log.Debugf("Found remote repository %s/%s for user %s",
					owner, name, user.Login)
			}
		}

		data := gin.H{
			"User": user,
			"Repo": repo,
		}

		// if we found a repository, we should display a page
		// to the user allowing them to activate.
		if repo != nil && len(repo.FullName) != 0 {
			// we should probably move this code to a
			// separate route, but for now we need to
			// add a CSRF token.
			data["Csrf"], _ = token.New(
				token.CsrfToken,
				user.Login,
			).Sign(user.Hash)

			c.HTML(http.StatusNotFound, "repo_activate.html", data)
		} else {
			c.HTML(http.StatusNotFound, "404.html", data)
		}

		c.Abort()
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

		if user != nil {
			// attempt to get the permissions from a local cache
			// just to avoid excess API calls to GitHub
			val, ok := c.Get("perm")
			if ok {
				c.Next()

				log.Debugf("%s using cached %+v permission to %s",
					user.Login, val, repo.FullName)
				return
			}
		}

		switch {
		// if the user is not authenticated, and the
		// repository is private, the user has NO permission
		// to view the repository.
		case user == nil && repo.IsPrivate == true:
			perm.Pull = false
			perm.Push = false
			perm.Admin = false

		// if the user is not authenticated, but the repository
		// is public, the user has pull-rights only.
		case user == nil && repo.IsPrivate == false:
			perm.Pull = true
			perm.Push = false
			perm.Admin = false

		case user.Admin:
			perm.Pull = true
			perm.Push = true
			perm.Admin = true

		// otherwise if the user is authenticated we should
		// check the remote system to get the users permissiosn.
		default:
			var err error
			perm, err = remote.FromContext(c).Perm(user, repo.Owner, repo.Name)
			if err != nil {
				perm.Pull = false
				perm.Push = false
				perm.Admin = false

				// debug
				log.Errorf("Error fetching permission for %s %s",
					user.Login, repo.FullName)
			}
			// if we couldn't fetch permissions, but the repository
			// is public, we should grant the user pull access.
			if err != nil && repo.IsPrivate == false {
				perm.Pull = true
			}
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
	repo := Repo(c)
	perm := Perm(c)

	if perm.Pull {
		c.Next()
		return
	}

	// if the user doesn't have pull permission to the
	// repository we display a 404 error to avoid leaking
	// repository information.
	c.HTML(http.StatusNotFound, "404.html", gin.H{
		"User": user,
		"Repo": repo,
		"Perm": perm,
	})

	c.Abort()
}

func MustPush(c *gin.Context) {
	user := User(c)
	repo := Repo(c)
	perm := Perm(c)

	// if the user has push access, immediately proceed
	// the middleware execution chain.
	if perm.Push {
		c.Next()
		return
	}

	data := gin.H{
		"User": user,
		"Repo": repo,
		"Perm": perm,
	}

	// if the user has pull access we should tell them
	// the operation is not authorized. Otherwise we should
	// give a 404 to avoid leaking information.
	if !perm.Pull {
		c.HTML(http.StatusNotFound, "404.html", data)
	} else {
		c.HTML(http.StatusUnauthorized, "401.html", data)
	}

	// debugging
	if user != nil {
		log.Debugf("%s denied write access to %s",
			user.Login, c.Request.URL.Path)

	} else {
		log.Debugf("Guest denied write access to %s %s",
			c.Request.Method,
			c.Request.URL.Path,
		)
	}

	c.Abort()
}
