package server

import (
	"encoding/base32"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/securecookie"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote"
	"github.com/drone/drone/router/middleware/session"
	"github.com/drone/drone/shared/httputil"
	"github.com/drone/drone/shared/token"
	"github.com/drone/drone/store"
)

func PostRepo(c *gin.Context) {
	remote := remote.FromContext(c)
	user := session.User(c)
	repo := session.Repo(c)

	if repo.IsActive {
		c.String(409, "Repository is already active.")
		return
	}

	if err := Config.Services.Limiter.LimitRepo(user, repo); err != nil {
		c.String(403, "Repository activation blocked by limiter")
		return
	}

	repo.IsActive = true
	repo.UserID = user.ID
	if !repo.AllowPush && !repo.AllowPull && !repo.AllowDeploy && !repo.AllowTag {
		repo.AllowPush = true
		repo.AllowPull = true
	}
	if repo.Visibility == "" {
		repo.Visibility = model.VisibilityPublic
		if repo.IsPrivate {
			repo.Visibility = model.VisibilityPrivate
		}
	}
	if repo.Config == "" {
		repo.Config = Config.Server.RepoConfig
	}
	if repo.Timeout == 0 {
		repo.Timeout = 60 // 1 hour default build time
	}
	if repo.Hash == "" {
		repo.Hash = base32.StdEncoding.EncodeToString(
			securecookie.GenerateRandomKey(32),
		)
	}

	// creates the jwt token used to verify the repository
	t := token.New(token.HookToken, repo.FullName)
	sig, err := t.Sign(repo.Hash)
	if err != nil {
		c.String(500, err.Error())
		return
	}

	link := fmt.Sprintf(
		"%s/hook?access_token=%s",
		httputil.GetURL(c.Request),
		sig,
	)

	err = remote.Activate(user, repo, link)
	if err != nil {
		c.String(500, err.Error())
		return
	}

	from, err := remote.Repo(user, repo.Owner, repo.Name)
	if err == nil {
		repo.Update(from)
	}

	err = store.UpdateRepo(c, repo)
	if err != nil {
		c.String(500, err.Error())
		return
	}

	c.JSON(200, repo)
}

func PatchRepo(c *gin.Context) {
	repo := session.Repo(c)
	user := session.User(c)

	in := new(model.RepoPatch)
	if err := c.Bind(in); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if (in.IsTrusted != nil || in.Timeout != nil || in.BuildCounter != nil) && !user.Admin {
		c.String(403, "Insufficient privileges")
		return
	}

	if in.AllowPush != nil {
		repo.AllowPush = *in.AllowPush
	}
	if in.AllowPull != nil {
		repo.AllowPull = *in.AllowPull
	}
	if in.AllowDeploy != nil {
		repo.AllowDeploy = *in.AllowDeploy
	}
	if in.AllowTag != nil {
		repo.AllowTag = *in.AllowTag
	}
	if in.IsGated != nil {
		repo.IsGated = *in.IsGated
	}
	if in.IsTrusted != nil {
		repo.IsTrusted = *in.IsTrusted
	}
	if in.Timeout != nil {
		repo.Timeout = *in.Timeout
	}
	if in.Config != nil {
		repo.Config = *in.Config
	}
	if in.Visibility != nil {
		switch *in.Visibility {
		case model.VisibilityInternal, model.VisibilityPrivate, model.VisibilityPublic:
			repo.Visibility = *in.Visibility
		default:
			c.String(400, "Invalid visibility type")
			return
		}
	}
	if in.BuildCounter != nil {
		repo.Counter = *in.BuildCounter
	}

	err := store.UpdateRepo(c, repo)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, repo)
}

func ChownRepo(c *gin.Context) {
	repo := session.Repo(c)
	user := session.User(c)
	repo.UserID = user.ID

	err := store.UpdateRepo(c, repo)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, repo)
}

func GetRepo(c *gin.Context) {
	c.JSON(http.StatusOK, session.Repo(c))
}

func DeleteRepo(c *gin.Context) {
	remove, _ := strconv.ParseBool(c.Query("remove"))
	remote := remote.FromContext(c)
	repo := session.Repo(c)
	user := session.User(c)

	repo.IsActive = false
	repo.UserID = 0

	err := store.UpdateRepo(c, repo)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if remove {
		err := store.DeleteRepo(c, repo)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}

	remote.Deactivate(user, repo, httputil.GetURL(c.Request))
	c.JSON(200, repo)
}

func RepairRepo(c *gin.Context) {
	remote := remote.FromContext(c)
	repo := session.Repo(c)
	user := session.User(c)

	// creates the jwt token used to verify the repository
	t := token.New(token.HookToken, repo.FullName)
	sig, err := t.Sign(repo.Hash)
	if err != nil {
		c.String(500, err.Error())
		return
	}

	// reconstruct the link
	host := httputil.GetURL(c.Request)
	link := fmt.Sprintf(
		"%s/hook?access_token=%s",
		host,
		sig,
	)

	remote.Deactivate(user, repo, host)
	err = remote.Activate(user, repo, link)
	if err != nil {
		c.String(500, err.Error())
		return
	}

	from, err := remote.Repo(user, repo.Owner, repo.Name)
	if err == nil {
		repo.Name = from.Name
		repo.Owner = from.Owner
		repo.FullName = from.FullName
		repo.Avatar = from.Avatar
		repo.Link = from.Link
		repo.Clone = from.Clone
		repo.IsPrivate = from.IsPrivate
		if repo.IsPrivate != from.IsPrivate {
			repo.ResetVisibility()
		}
		store.UpdateRepo(c, repo)
	}

	c.Writer.WriteHeader(http.StatusOK)
}

func MoveRepo(c *gin.Context) {
	remote := remote.FromContext(c)
	repo := session.Repo(c)
	user := session.User(c)

	to, exists := c.GetQuery("to")
	if !exists {
		err := fmt.Errorf("Missing required to query value")
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	owner, name, errParse := model.ParseRepo(to)
	if errParse != nil {
		c.AbortWithError(http.StatusInternalServerError, errParse)
		return
	}

	from, err := remote.Repo(user, owner, name)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if !from.Perm.Admin {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	repo.Name = from.Name
	repo.Owner = from.Owner
	repo.FullName = from.FullName
	repo.Avatar = from.Avatar
	repo.Link = from.Link
	repo.Clone = from.Clone
	repo.IsPrivate = from.IsPrivate
	if repo.IsPrivate != from.IsPrivate {
		repo.ResetVisibility()
	}

	errStore := store.UpdateRepo(c, repo)
	if errStore != nil {
		c.AbortWithError(http.StatusInternalServerError, errStore)
		return
	}

	// creates the jwt token used to verify the repository
	t := token.New(token.HookToken, repo.FullName)
	sig, err := t.Sign(repo.Hash)
	if err != nil {
		c.String(500, err.Error())
		return
	}

	// reconstruct the link
	host := httputil.GetURL(c.Request)
	link := fmt.Sprintf(
		"%s/hook?access_token=%s",
		host,
		sig,
	)

	remote.Deactivate(user, repo, host)
	err = remote.Activate(user, repo, link)
	if err != nil {
		c.String(500, err.Error())
		return
	}
	c.Writer.WriteHeader(http.StatusOK)
}
