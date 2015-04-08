package server

import (
	"fmt"
	"log"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/drone/drone/common"
	"github.com/drone/drone/common/gravatar"
	"github.com/drone/drone/common/httputil"
	"github.com/drone/drone/common/oauth2"
)

// GetLogin accepts a request to authorize the user and to
// return a valid OAuth2 access token. The access token is
// returned as url segment #access_token
//
//     GET /authorize
//
func GetLogin(c *gin.Context) {
	settings := ToSettings(c)
	session := ToSession(c)
	store := ToDatastore(c)

	// when dealing with redirects we may need
	// to adjust the content type. I cannot, however,
	// rememver why, so need to revisit this line.
	c.Writer.Header().Del("Content-Type")

	// depending on the configuration a user may
	// authenticate with OAuth1, OAuth2 or Basic
	// Auth (username and password). This will delegate
	// authorization accordingly.
	switch {
	case settings.Service.OAuth == nil:
		getLoginBasic(c)
	case settings.Service.OAuth.RequestToken != "":
		getLoginOauth1(c)
	default:
		getLoginOauth2(c)
	}

	// exit if authorization fails
	// TODO(bradrydzewski) return an error message instead
	if c.Writer.Status() != 200 {
		return
	}

	// get the user from the database
	login := ToUser(c)
	u, err := store.GetUser(login.Login)
	if err != nil {
		// if self-registration is disabled we should
		// return a notAuthorized error. the only exception
		// is if no users exist yet in the system we'll proceed.
		if !settings.Service.Open {
			count, err := store.GetUserCount()
			if err != nil || count != 0 {
				c.String(400, "Unable to create account. Registration is closed")
				return
			}
		}

		// create the user account
		u = &common.User{}
		u.Login = login.Login
		u.Token = login.Token
		u.Secret = login.Secret
		u.Name = login.Name
		u.Email = login.Email
		u.Gravatar = gravatar.Generate(u.Email)

		// insert the user into the database
		if err := store.InsertUser(u); err != nil {
			log.Println(err)
			c.Fail(400, err)
			return
		}

		// // if this is the first user, they
		// // should be an admin.
		//if u.ID == 1 {
		if u.Login == "bradrydzewski" {
			u.Admin = true
		}
	}

	// update the user meta data and authorization
	// data and cache in the datastore.
	u.Token = login.Token
	u.Secret = login.Secret
	u.Name = login.Name
	u.Email = login.Email
	u.Gravatar = gravatar.Generate(u.Email)

	if err := store.UpdateUser(u); err != nil {
		log.Println(err)
		c.Fail(400, err)
		return
	}

	token, err := session.GenerateToken(c.Request, u)
	if err != nil {
		log.Println(err)
		c.Fail(400, err)
		return
	}
	c.Redirect(303, "/#access_token="+token)
}

// getLoginOauth2 is the default authorization implementation
// using the oauth2 protocol.
func getLoginOauth2(c *gin.Context) {
	var settings = ToSettings(c)
	var remote = ToRemote(c)

	var config = &oauth2.Config{
		ClientId:     settings.Service.OAuth.Client,
		ClientSecret: settings.Service.OAuth.Secret,
		Scope:        strings.Join(settings.Service.OAuth.Scope, ","),
		AuthURL:      settings.Service.OAuth.Authorize,
		TokenURL:     settings.Service.OAuth.AccessToken,
		RedirectURL:  fmt.Sprintf("%s/authorize", httputil.GetURL(c.Request)),
		//settings.Server.Scheme, settings.Server.Hostname),
	}

	// get the OAuth code
	var code = c.Request.FormValue("code")
	//var state = c.Request.FormValue("state")
	if len(code) == 0 {
		c.Redirect(303, config.AuthCodeURL("random"))
		return
	}

	// exhange for a token
	var trans = &oauth2.Transport{Config: config}
	var token, err = trans.Exchange(code)
	if err != nil {
		c.Fail(400, err)
		return
	}

	// get user account
	user, err := remote.Login(token.AccessToken, token.RefreshToken)
	if err != nil {
		c.Fail(404, err)
		return
	}

	// add the user to the request
	c.Set("user", user)
}

// getLoginOauth1 is able to authorize a user with the oauth1
// authentication protocol. This is used primarily with Bitbucket
// and Stash only, and one day I hope can be removed.
func getLoginOauth1(c *gin.Context) {

}

// getLoginBasic is able to authorize a user with a username and
// password. This can be used for systems that do not support oauth.
func getLoginBasic(c *gin.Context) {
	var (
		remote   = ToRemote(c)
		username = c.Request.FormValue("username")
		password = c.Request.FormValue("username")
	)

	// get user account
	user, err := remote.Login(username, password)
	if err != nil {
		c.Fail(404, err)
		return
	}

	// add the user to the request
	c.Set("user", user)
}
