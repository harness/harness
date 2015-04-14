package server

import (
	"github.com/drone/drone/common"
	"github.com/gin-gonic/gin"
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

	err := store.DeleteSubscriber(user.Login, repo.FullName)
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

	err := store.InsertSubscriber(user.Login, repo.FullName)
	if err != nil {
		c.Fail(400, err)
	} else {
		c.JSON(200, &common.Subscriber{Subscribed: true})
	}
}
