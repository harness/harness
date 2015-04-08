package server

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/drone/drone/common"
	"github.com/drone/drone/common/gravatar"
)

// GetUserCurr accepts a request to retrieve the
// currently authenticated user from the datastore
// and return in JSON format.
//
//     GET /api/user
//
func GetUserCurr(c *gin.Context) {
	c.JSON(200, ToUser(c))
}

// PutUserCurr accepts a request to update the currently
// authenticated User profile.
//
//     PUT /api/user
//
func PutUserCurr(c *gin.Context) {
	ds := ToDatastore(c)
	me := ToUser(c)

	in := &common.User{}
	if !c.BindWith(in, binding.JSON) {
		return
	}
	me.Email = in.Email
	me.Gravatar = gravatar.Generate(in.Email)
	err := ds.UpdateUser(me)
	if err != nil {
		c.Fail(400, err)
	} else {
		c.JSON(200, me)
	}
}

// GetUserRepos accepts a request to get the currently
// authenticated user's repository list from the datastore,
// encoded and returned in JSON format.
//
//     GET /api/user/repos
//
func GetUserRepos(c *gin.Context) {
	ds := ToDatastore(c)
	me := ToUser(c)
	repos, err := ds.GetUserRepos(me.Login)
	if err != nil {
		c.Fail(400, err)
	} else {
		c.JSON(200, &repos)
	}
}
