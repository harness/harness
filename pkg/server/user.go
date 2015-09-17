package server

import (
	// "crypto/sha1"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/gin-gonic/gin"
	"github.com/drone/drone/Godeps/_workspace/src/github.com/gin-gonic/gin/binding"
	"github.com/drone/drone/Godeps/_workspace/src/github.com/ungerik/go-gravatar"
	log "github.com/drone/drone/Godeps/_workspace/src/github.com/Sirupsen/logrus"

	"github.com/drone/drone/pkg/token"
	"github.com/drone/drone/pkg/types"
)

// GetUserCurr accepts a request to retrieve the
// currently authenticated user from the datastore
// and return in JSON format.
//
//     GET /api/user
//
func GetUserCurr(c *gin.Context) {
	u := ToUser(c)
	// f := fmt.Printf("% x", sha1.Sum(u.Hash))

	// v := struct {
	// 	*types.User

	// 	// token fingerprint
	// 	Token string `json:"token"`
	// }{u, f}
	tokenstr, err := generateUserToken(u)
	if err != nil {
		log.Errorf("cannot create token for %s. %s", u.Login, err)
		tokenstr = ""
	}
	userWithToken := struct {
		*types.User
		Token string `json:"token,omitempty"`
	}{u, tokenstr}

	c.JSON(200, userWithToken)
}

// PutUserCurr accepts a request to update the currently
// authenticated User profile.
//
//     PUT /api/user
//
func PutUserCurr(c *gin.Context) {
	store := ToDatastore(c)
	user := ToUser(c)

	in := &types.User{}
	if !c.BindWith(in, binding.JSON) {
		return
	}
	// TODO: we are no longer auto-generating avatar
	user.Email = in.Email
	user.Avatar = gravatar.Hash(in.Email)
	err := store.SetUser(user)
	if err != nil {
		c.Fail(400, err)
	} else {
		c.JSON(200, user)
	}
}

// GetUserRepos accepts a request to get the currently
// authenticated user's repository list from the datastore,
// encoded and returned in JSON format.
//
//     GET /api/user/repos
//
func GetUserRepos(c *gin.Context) {
	store := ToDatastore(c)
	user := ToUser(c)
	repos, err := store.RepoList(user)
	if err != nil {
		c.Fail(400, err)
	} else {
		c.JSON(200, &repos)
	}
}

// GetUserFeed accepts a request to get the currently
// authenticated user's build feed from the datastore,
// encoded and returned in JSON format.
//
//     GET /api/user/feed
//
func GetUserFeed(c *gin.Context) {
	store := ToDatastore(c)
	user := ToUser(c)
	feed, err := store.UserFeed(user, 25, 0)
	if err != nil {
		c.Fail(400, err)
	} else {
		c.JSON(200, &feed)
	}
}

// POST /api/user/token
func PostUserToken(c *gin.Context) {
	user := ToUser(c)

	token := token.New(token.UserToken, user.Login)
	tokenstr, err := token.Sign(user.Hash)
	if err != nil {
		c.Fail(500, err)
	} else {
		c.String(200, tokenstr)
	}
}
