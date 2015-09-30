package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/CiscoCloud/drone/model"
	"github.com/CiscoCloud/drone/router/middleware/context"
	"github.com/CiscoCloud/drone/router/middleware/session"
	"github.com/CiscoCloud/drone/shared/token"
)

func GetNodes(c *gin.Context) {
	db := context.Database(c)
	nodes, err := model.GetNodeList(db)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
	} else {
		c.IndentedJSON(http.StatusOK, nodes)
	}
}

func ShowNodes(c *gin.Context) {
	db := context.Database(c)
	user := session.User(c)
	nodes, _ := model.GetNodeList(db)
	token, _ := token.New(token.CsrfToken, user.Login).Sign(user.Hash)
	c.HTML(http.StatusOK, "nodes.html", gin.H{"User": user, "Nodes": nodes, "Csrf": token})
}

func GetNode(c *gin.Context) {

}

func PostNode(c *gin.Context) {
	db := context.Database(c)
	engine := context.Engine(c)

	node := &model.Node{}
	err := c.Bind(node)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	node.Arch = "linux_amd64"

	err = model.InsertNode(db, node)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	ok := engine.Allocate(node)
	if !ok {
		c.AbortWithStatus(http.StatusInternalServerError)
	} else {
		c.IndentedJSON(http.StatusOK, node)
	}

}

func DeleteNode(c *gin.Context) {
	db := context.Database(c)
	engine := context.Engine(c)

	id, _ := strconv.Atoi(c.Param("node"))
	node, err := model.GetNode(db, int64(id))
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	err = model.DeleteNode(db, node)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	engine.Deallocate(node)
}
