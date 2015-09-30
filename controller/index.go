package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/CiscoCloud/drone/model"
	"github.com/CiscoCloud/drone/router/middleware/context"
	"github.com/CiscoCloud/drone/router/middleware/session"
	"github.com/CiscoCloud/drone/shared/httputil"
	"github.com/CiscoCloud/drone/shared/token"
)

func ShowIndex(c *gin.Context) {
	// remote := context.Remote(c)
	user := session.User(c)
	if user == nil {
		c.HTML(200, "login.html", gin.H{})
		return
	}

	// attempt to get the repository list from the
	// cache since the operation is expensive
	// v, ok := cache.Get(user.Login)
	// if ok {
	// 	c.HTML(200, "repos.html", gin.H{
	// 		"User":  user,
	// 		"Repos": v,
	// 	})
	// 	return
	// }

	// fetch the repmote repos
	// repos, err := remote.Repos(user)
	// if err != nil {
	// 	c.AbortWithStatus(http.StatusInternalServerError)
	// 	return
	// }
	// cache.Add(user.Login, repos)

	c.HTML(200, "repos.html", gin.H{
		"User": user,
		// "Repos": repos,
	})
}

func ShowLogin(c *gin.Context) {
	c.HTML(200, "login.html", gin.H{"Error": c.Query("error")})
}

func ShowUser(c *gin.Context) {
	user := session.User(c)
	token, _ := token.New(
		token.CsrfToken,
		user.Login,
	).Sign(user.Hash)

	c.HTML(200, "user.html", gin.H{
		"User": user,
		"Csrf": token,
	})
}

func ShowUsers(c *gin.Context) {
	db := context.Database(c)
	user := session.User(c)
	if !user.Admin {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	users, _ := model.GetUserList(db)

	token, _ := token.New(
		token.CsrfToken,
		user.Login,
	).Sign(user.Hash)

	c.HTML(200, "users.html", gin.H{
		"User":  user,
		"Users": users,
		"Csrf":  token,
	})
}

func ShowRepo(c *gin.Context) {
	db := context.Database(c)
	user := session.User(c)
	repo := session.Repo(c)
	if !user.Admin {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	builds, _ := model.GetBuildList(db, repo)
	groups := []*model.BuildGroup{}

	var curr *model.BuildGroup
	for _, build := range builds {
		date := time.Unix(build.Created, 0).Format("Jan 2 2006")
		if curr == nil || curr.Date != date {
			curr = &model.BuildGroup{}
			curr.Date = date
			groups = append(groups, curr)
		}
		curr.Builds = append(curr.Builds, build)
	}

	httputil.SetCookie(c.Writer, c.Request, "user_last", repo.FullName)

	c.HTML(200, "repo.html", gin.H{
		"User":   user,
		"Repo":   repo,
		"Builds": builds,
		"Groups": groups,
	})

}

func ShowRepoConf(c *gin.Context) {
	db := context.Database(c)
	user := session.User(c)
	repo := session.Repo(c)
	key, _ := model.GetKey(db, repo)
	if !user.Admin {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	var view = "repo_config.html"
	switch c.Param("action") {
	case "delete":
		view = "repo_delete.html"
	case "encrypt":
		view = "repo_secret.html"
	case "badges":
		view = "repo_badge.html"
	}

	token, _ := token.New(
		token.CsrfToken,
		user.Login,
	).Sign(user.Hash)

	c.HTML(200, view, gin.H{
		"User": user,
		"Repo": repo,
		"Key":  key,
		"Csrf": token,
		"Link": httputil.GetURL(c.Request),
	})
}

func ShowBuild(c *gin.Context) {
	db := context.Database(c)
	user := session.User(c)
	repo := session.Repo(c)
	num, _ := strconv.Atoi(c.Param("number"))
	seq, _ := strconv.Atoi(c.Param("job"))
	if seq == 0 {
		seq = 1
	}

	build, err := model.GetBuildNumber(db, repo, num)
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	jobs, err := model.GetJobList(db, build)
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	var job *model.Job
	for _, j := range jobs {
		if j.Number == seq {
			job = j
			break
		}
	}

	httputil.SetCookie(c.Writer, c.Request, "user_last", repo.FullName)

	token, _ := token.New(
		token.CsrfToken,
		user.Login,
	).Sign(user.Hash)

	c.HTML(200, "build.html", gin.H{
		"User":  user,
		"Repo":  repo,
		"Build": build,
		"Jobs":  jobs,
		"Job":   job,
		"Csrf":  token,
	})
}
