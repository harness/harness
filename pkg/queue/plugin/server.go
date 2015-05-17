package plugin

import (
	"net/http"

	"github.com/drone/drone/pkg/queue"
	"github.com/gin-gonic/gin"
)

// Handle returns an http.Handler that enables a remote
// client to interop with a Queue over http.
func Handle(queue queue.Queue, token string) http.Handler {
	r := gin.New()

	// middleware to validate the authorization token
	// and to inject the queue into the http context.
	bearer := "Bearer " + token
	r.Use(func(c *gin.Context) {
		if c.Request.Header.Get("Authorization") != bearer {
			c.AbortWithStatus(403)
			return
		}
		c.Set("queue", queue)
		c.Next()
	})

	r.POST("/queue", publish)
	r.DELETE("/queue", remove)
	r.POST("/queue/pull", pull)
	r.POST("/queue/ack", ack)
	r.POST("/queue/items", items)

	return r
}

// publish handles an http request to the queue
// to insert work at the tail.
func publish(c *gin.Context) {
	q := fromContext(c)
	work := &queue.Work{}
	if !c.Bind(work) {
		c.AbortWithStatus(400)
		return
	}
	err := q.Publish(work)
	if err != nil {
		c.Fail(500, err)
		return
	}
	c.Writer.WriteHeader(200)
}

// remove handles an http request to the queue
// to remove a work item.
func remove(c *gin.Context) {
	q := fromContext(c)
	work := &queue.Work{}
	if !c.Bind(work) {
		c.AbortWithStatus(400)
		return
	}
	err := q.Remove(work)
	if err != nil {
		c.Fail(500, err)
		return
	}
	c.Writer.WriteHeader(200)
}

// pull handles an http request to the queue
// to retrieve work.
func pull(c *gin.Context) {
	q := fromContext(c)
	work := q.PullClose(c.Writer)
	if work == nil {
		c.AbortWithStatus(500)
		return
	}
	c.JSON(200, work)
}

// ack handles an http request to the queue
// to confirm an item was successfully pulled.
func ack(c *gin.Context) {
	q := fromContext(c)
	work := &queue.Work{}
	if !c.Bind(work) {
		c.AbortWithStatus(400)
		return
	}
	err := q.Ack(work)
	if err != nil {
		c.Fail(500, err)
		return
	}
	c.Writer.WriteHeader(200)
}

// items handles an http request to the queue to
// return a list of all work items.
func items(c *gin.Context) {
	q := fromContext(c)
	items := q.Items()
	c.JSON(200, items)
}

// helper function to retrieve the Queue from
// the context and cast appropriately.
func fromContext(c *gin.Context) queue.Queue {
	return c.MustGet("queue").(queue.Queue)
}
