package server

import (
	"github.com/drone/drone/Godeps/_workspace/src/github.com/gin-gonic/gin"
)

func GetQueue(c *gin.Context) {
	queue := ToQueue(c)
	items := queue.Items()
	c.JSON(200, items)
}
