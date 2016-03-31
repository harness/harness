package api

import (
	"io/ioutil"

	"github.com/drone/drone/router/middleware/session"

	"github.com/gin-gonic/gin"
	"github.com/square/go-jose"
)

func Sign(c *gin.Context) {
	repo := session.Repo(c)

	in, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.String(400, "Unable to read request body. %s.", err.Error())
		return
	}

	signer, err := jose.NewSigner(jose.HS256, []byte(repo.Hash))
	if err != nil {
		c.String(500, "Unable to create the signer. %s.", err.Error())
		return
	}

	signed, err := signer.Sign(in)
	if err != nil {
		c.String(500, "Unable to sign input. %s", err.Error())
		return
	}

	out, err := signed.CompactSerialize()
	if err != nil {
		c.String(500, "Unable to serialize signature. %s", err.Error())
		return
	}

	c.String(200, out)
}
