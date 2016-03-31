package web

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	log "github.com/Sirupsen/logrus"
	"github.com/drone/drone/model"
	"github.com/drone/drone/remote"
	"github.com/drone/drone/shared/crypto"
	"github.com/drone/drone/shared/httputil"
	"github.com/drone/drone/shared/token"
	"github.com/drone/drone/store"
)

func GetLogin(c *gin.Context) {
	remote := remote.FromContext(c)

	// when dealing with redirects we may need
	// to adjust the content type. I cannot, however,
	// remember why, so need to revisit this line.
	c.Writer.Header().Del("Content-Type")

	tmpuser, open, err := remote.Login(c.Writer, c.Request)
	if err != nil {
		log.Errorf("cannot authenticate user. %s", err)
		c.Redirect(303, "/login?error=oauth_error")
		return
	}
	// this will happen when the user is redirected by
	// the remote provide as part of the oauth dance.
	if tmpuser == nil {
		return
	}

	// get the user from the database
	u, err := store.GetUserLogin(c, tmpuser.Login)
	if err != nil {
		count, err := store.GetUserCount(c)
		if err != nil {
			log.Errorf("cannot register %s. %s", tmpuser.Login, err)
			c.Redirect(303, "/login?error=internal_error")
			return
		}

		// if self-registration is disabled we should
		// return a notAuthorized error. the only exception
		// is if no users exist yet in the system we'll proceed.
		if !open && count != 0 {
			log.Errorf("cannot register %s. registration closed", tmpuser.Login)
			c.Redirect(303, "/login?error=access_denied")
			return
		}

		// create the user account
		u = &model.User{}
		u.Login = tmpuser.Login
		u.Token = tmpuser.Token
		u.Secret = tmpuser.Secret
		u.Email = tmpuser.Email
		u.Avatar = tmpuser.Avatar
		u.Hash = crypto.Rand()

		// insert the user into the database
		if err := store.CreateUser(c, u); err != nil {
			log.Errorf("cannot insert %s. %s", u.Login, err)
			c.Redirect(303, "/login?error=internal_error")
			return
		}

		// if this is the first user, they
		// should be an admin.
		if count == 0 {
			u.Admin = true
		}
	}

	// update the user meta data and authorization
	// data and cache in the datastore.
	u.Token = tmpuser.Token
	u.Secret = tmpuser.Secret
	u.Email = tmpuser.Email
	u.Avatar = tmpuser.Avatar

	if err := store.UpdateUser(c, u); err != nil {
		log.Errorf("cannot update %s. %s", u.Login, err)
		c.Redirect(303, "/login?error=internal_error")
		return
	}

	exp := time.Now().Add(time.Hour * 72).Unix()
	token := token.New(token.SessToken, u.Login)
	tokenstr, err := token.SignExpires(u.Hash, exp)
	if err != nil {
		log.Errorf("cannot create token for %s. %s", u.Login, err)
		c.Redirect(303, "/login?error=internal_error")
		return
	}

	httputil.SetCookie(c.Writer, c.Request, "user_sess", tokenstr)
	redirect := httputil.GetCookie(c.Request, "user_last")
	if len(redirect) == 0 {
		redirect = "/"
	}
	c.Redirect(303, redirect)

}

func GetLogout(c *gin.Context) {

	httputil.DelCookie(c.Writer, c.Request, "user_sess")
	httputil.DelCookie(c.Writer, c.Request, "user_last")
	c.Redirect(303, "/login")
}

func GetLoginToken(c *gin.Context) {
	remote := remote.FromContext(c)

	in := &tokenPayload{}
	err := c.Bind(in)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	login, err := remote.Auth(in.Access, in.Refresh)
	if err != nil {
		c.AbortWithError(http.StatusUnauthorized, err)
		return
	}

	user, err := store.GetUserLogin(c, login)
	if err != nil {
		c.AbortWithError(http.StatusNotFound, err)
		return
	}

	exp := time.Now().Add(time.Hour * 72).Unix()
	token := token.New(token.SessToken, user.Login)
	tokenstr, err := token.SignExpires(user.Hash, exp)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.IndentedJSON(http.StatusOK, &tokenPayload{
		Access:  tokenstr,
		Expires: exp - time.Now().Unix(),
	})
}

type tokenPayload struct {
	Access  string `json:"access_token,omitempty"`
	Refresh string `json:"refresh_token,omitempty"`
	Expires int64  `json:"expires_in,omitempty"`
}
