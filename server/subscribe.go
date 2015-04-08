package server

import (
	"github.com/gin-gonic/gin"

	"github.com/drone/drone/common"
)

// GetSubscriber accepts a request to retrieve a repository
// subscriber from the datastore for the given repository by
// user Login.
//
//     GET /api/subscribers/:owner/:name/:login
//
func GetSubscriber(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	login := c.Params.ByName("login")
	subsc, err := store.GetSubscriber(repo.FullName, login)
	if err != nil {
		c.Fail(404, err)
	} else {
		c.JSON(200, subsc)
	}
}

// GetSubscribers accepts a request to retrieve a repository
// watchers from the datastore for the given repository.
//
//     GET /api/subscribers/:owner/:name
//
func GetSubscribers(c *gin.Context) {
	// store := ToDatastore(c)
	// repo := ToRepo(c)
	// subs, err := store.GetSubscribers(repo.FullName)
	// if err != nil {
	// 	c.Fail(404, err)
	// } else {
	// 	c.JSON(200, subs)
	// }
	c.Writer.WriteHeader(501)
}

// Unubscribe accapets a request to unsubscribe the
// currently authenticated user to the repository.
//
//     DEL /api/subscribers/:owner/:name
//
func Unsubscribe(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	user := ToUser(c)
	sub, err := store.GetSubscriber(repo.FullName, user.Login)
	if err != nil {
		c.Fail(404, err)
	}
	err = store.DeleteSubscriber(repo.FullName, sub)
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
	subscriber := &common.Subscriber{
		Login:      user.Login,
		Subscribed: true,
	}
	err := store.InsertSubscriber(repo.FullName, subscriber)
	if err != nil {
		c.Fail(400, err)
	} else {
		c.JSON(200, subscriber)
	}
}
