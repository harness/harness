package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/drone/drone/model"
	"github.com/drone/drone/router/middleware/context"
	"github.com/drone/drone/router/middleware/session"
	"github.com/drone/drone/shared/token"
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
	env := context.Envconfig(c)
	db := context.Database(c)
	user := session.User(c)

	nodes, _ := model.GetNodeList(db)
	token, _ := token.New(token.CsrfToken, user.Login).Sign(user.Hash)

	c.HTML(http.StatusOK, "nodes.html", gin.H{
		"Root": env.String("SERVER_ROOT", ""),
		"User": user,
		"Nodes": nodes,
		"Csrf": token,
	})
}

func GetNode(c *gin.Context) {

}

func PostNode(c *gin.Context) {
	db := context.Database(c)
	engine := context.Engine(c)

	in := struct {
		Addr string `json:"address"`
		Arch string `json:"architecture"`
		Cert string `json:"cert"`
		Key  string `json:"key"`
		CA   string `json:"ca"`
	}{}
	err := c.Bind(&in)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	node := &model.Node{}
	node.Addr = in.Addr
	node.Cert = in.Cert
	node.Key = in.Key
	node.CA = in.CA
	node.Arch = "linux_amd64"

	err = engine.Allocate(node)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	err = model.InsertNode(db, node)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.IndentedJSON(http.StatusOK, node)
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
