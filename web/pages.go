package web

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/drone/drone/cache"
	"github.com/drone/drone/model"
	"github.com/drone/drone/router/middleware/session"
	"github.com/drone/drone/shared/httputil"
	"github.com/drone/drone/shared/token"
	"github.com/drone/drone/store"
)

func ShowIndex(c *gin.Context) {
	user := session.User(c)
	if user == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// get the repository list from the cache
	repos, err := cache.GetRepos(c, user)
	if err != nil {
		c.String(400, err.Error())
		return
	}

	// filter to only show the currently active ones
	activeRepos, err := store.GetRepoListOf(c,repos)
	if err != nil {
		c.String(400, err.Error())
		return
	}

	c.HTML(200, "index.html", gin.H{
		"User":  user,
		"Repos": activeRepos,
	})
}

func ShowAllRepos(c *gin.Context) {
	user := session.User(c)
	if user == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	// get the repository list from the cache
	repos, err := cache.GetRepos(c, user)
	if err != nil {
		c.String(400, err.Error())
		return
	}

	c.HTML(200, "repos.html", gin.H{
		"User":  user,
		"Repos": repos,
	})
}

func ShowLogin(c *gin.Context) {
	c.HTML(200, "login.html", gin.H{"Error": c.Query("error")})
}

func ShowLoginForm(c *gin.Context) {
	c.HTML(200, "login_form.html", gin.H{})
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
	user := session.User(c)
	if !user.Admin {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	users, _ := store.GetUserList(c)

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
	user := session.User(c)
	repo := session.Repo(c)

	builds, _ := store.GetBuildList(c, repo)
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

	user := session.User(c)
	repo := session.Repo(c)
	key, _ := store.GetKey(c, repo)

	token, _ := token.New(
		token.CsrfToken,
		user.Login,
	).Sign(user.Hash)

	c.HTML(200, "repo_config.html", gin.H{
		"User": user,
		"Repo": repo,
		"Key":  key,
		"Csrf": token,
		"Link": httputil.GetURL(c.Request),
	})
}

func ShowRepoEncrypt(c *gin.Context) {
	user := session.User(c)
	repo := session.Repo(c)

	token, _ := token.New(
		token.CsrfToken,
		user.Login,
	).Sign(user.Hash)

	c.HTML(200, "repo_secret.html", gin.H{
		"User": user,
		"Repo": repo,
		"Csrf": token,
	})
}

func ShowRepoBadges(c *gin.Context) {
	user := session.User(c)
	repo := session.Repo(c)

	c.HTML(200, "repo_badge.html", gin.H{
		"User": user,
		"Repo": repo,
		"Link": httputil.GetURL(c.Request),
	})
}

func ShowBuild(c *gin.Context) {
	user := session.User(c)
	repo := session.Repo(c)
	num, _ := strconv.Atoi(c.Param("number"))
	seq, _ := strconv.Atoi(c.Param("job"))
	if seq == 0 {
		seq = 1
	}

	build, err := store.GetBuildNumber(c, repo, num)
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	jobs, err := store.GetJobList(c, build)
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

	var csrf string
	if user != nil {
		csrf, _ = token.New(
			token.CsrfToken,
			user.Login,
		).Sign(user.Hash)
	}

	c.HTML(200, "build.html", gin.H{
		"User":  user,
		"Repo":  repo,
		"Build": build,
		"Jobs":  jobs,
		"Job":   job,
		"Csrf":  csrf,
	})
}

func ShowNodes(c *gin.Context) {
	user := session.User(c)
	nodes, _ := store.GetNodeList(c)
	token, _ := token.New(token.CsrfToken, user.Login).Sign(user.Hash)
	c.HTML(http.StatusOK, "nodes.html", gin.H{"User": user, "Nodes": nodes, "Csrf": token})
}
