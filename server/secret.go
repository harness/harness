package server

import (
	"net/http"

	"github.com/drone/drone/model"
	"github.com/drone/drone/router/middleware/session"

	"github.com/gin-gonic/gin"
)

// GetSecret gets the named secret from the database and writes
// to the response in json format.
func GetSecret(c *gin.Context) {
	var (
		repo = session.Repo(c)
		name = c.Param("secret")
	)
	secret, err := Config.Services.Secrets.SecretFind(repo, name)
	if err != nil {
		c.String(404, "Error getting secret %q. %s", name, err)
		return
	}
	c.JSON(200, secret.Copy())
}

// PostSecret persists the secret to the database.
func PostSecret(c *gin.Context) {
	repo := session.Repo(c)

	in := new(model.Secret)
	if err := c.Bind(in); err != nil {
		c.String(http.StatusBadRequest, "Error parsing secret. %s", err)
		return
	}
	secret := &model.Secret{
		RepoID: repo.ID,
		Name:   in.Name,
		Value:  in.Value,
		Events: in.Events,
		Images: in.Images,
	}
	if err := secret.Validate(); err != nil {
		c.String(400, "Error inserting secret. %s", err)
		return
	}
	if err := Config.Services.Secrets.SecretCreate(repo, secret); err != nil {
		c.String(500, "Error inserting secret %q. %s", in.Name, err)
		return
	}
	c.JSON(200, secret.Copy())
}

// PatchSecret updates the secret in the database.
func PatchSecret(c *gin.Context) {
	var (
		repo = session.Repo(c)
		name = c.Param("secret")
	)

	in := new(model.Secret)
	err := c.Bind(in)
	if err != nil {
		c.String(http.StatusBadRequest, "Error parsing secret. %s", err)
		return
	}

	secret, err := Config.Services.Secrets.SecretFind(repo, name)
	if err != nil {
		c.String(404, "Error getting secret %q. %s", name, err)
		return
	}
	if in.Value != "" {
		secret.Value = in.Value
	}
	if len(in.Events) != 0 {
		secret.Events = in.Events
	}
	if len(in.Images) != 0 {
		secret.Images = in.Images
	}

	if err := secret.Validate(); err != nil {
		c.String(400, "Error updating secret. %s", err)
		return
	}
	if err := Config.Services.Secrets.SecretUpdate(repo, secret); err != nil {
		c.String(500, "Error updating secret %q. %s", in.Name, err)
		return
	}
	c.JSON(200, secret.Copy())
}

// GetSecretList gets the secret list from the database and writes
// to the response in json format.
func GetSecretList(c *gin.Context) {
	repo := session.Repo(c)
	list, err := Config.Services.Secrets.SecretList(repo)
	if err != nil {
		c.String(500, "Error getting secret list. %s", err)
		return
	}
	// copy the secret detail to remove the sensitive
	// password and token fields.
	for i, secret := range list {
		list[i] = secret.Copy()
	}
	c.JSON(200, list)
}

// DeleteSecret deletes the named secret from the database.
func DeleteSecret(c *gin.Context) {
	var (
		repo = session.Repo(c)
		name = c.Param("secret")
	)
	if err := Config.Services.Secrets.SecretDelete(repo, name); err != nil {
		c.String(500, "Error deleting secret %q. %s", name, err)
		return
	}
	c.String(204, "")
}
