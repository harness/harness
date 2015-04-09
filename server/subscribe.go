package server

import (
	"github.com/gin-gonic/gin"

	"github.com/drone/drone/common"
)

// Unubscribe accapets a request to unsubscribe the
// currently authenticated user to the repository.
//
//     DEL /api/subscribers/:owner/:name
//
func Unsubscribe(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	user := ToUser(c)

	delete(user.Repos, repo.FullName)
	err := store.UpdateUser(user)
	if err != nil {
		c.Fail(400, err)
	} else {
		c.Writer.WriteHeader(200)
	}
}

// Subscribe accapets a request to subscribe the
// currently authenticated user to the repository.
//
//     POST /api/subscriber/:owner/:name
//
func Subscribe(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	user := ToUser(c)
	if user.Repos == nil {
		user.Repos = map[string]struct{}{}
	}
	user.Repos[repo.FullName] = struct{}{}
	err := store.UpdateUser(user)
	if err != nil {
		c.Fail(400, err)
	} else {
		c.JSON(200, &common.Subscriber{Subscribed: true})
	}
}
