package server

import (
	"net/http"

	"github.com/drone/drone/model"
	"github.com/drone/drone/router/middleware/session"

	"github.com/gin-gonic/gin"
)

// GetRegistry gets the name registry from the database and writes
// to the response in json format.
func GetRegistry(c *gin.Context) {
	var (
		repo = session.Repo(c)
		name = c.Param("registry")
	)
	registry, err := Config.Services.Registries.RegistryFind(repo, name)
	if err != nil {
		c.String(404, "Error getting registry %q. %s", name, err)
		return
	}
	c.JSON(200, registry.Copy())
}

// PostRegistry persists the registry to the database.
func PostRegistry(c *gin.Context) {
	repo := session.Repo(c)

	in := new(model.Registry)
	if err := c.Bind(in); err != nil {
		c.String(http.StatusBadRequest, "Error parsing request. %s", err)
		return
	}
	registry := &model.Registry{
		RepoID:   repo.ID,
		Address:  in.Address,
		Username: in.Username,
		Password: in.Password,
		Token:    in.Token,
		Email:    in.Email,
	}
	if err := registry.Validate(); err != nil {
		c.String(400, "Error inserting registry. %s", err)
		return
	}
	if err := Config.Services.Registries.RegistryCreate(repo, registry); err != nil {
		c.String(500, "Error inserting registry %q. %s", in.Address, err)
		return
	}
	c.JSON(200, in.Copy())
}

// PatchRegistry updates the registry in the database.
func PatchRegistry(c *gin.Context) {
	var (
		repo = session.Repo(c)
		name = c.Param("registry")
	)

	in := new(model.Registry)
	err := c.Bind(in)
	if err != nil {
		c.String(http.StatusBadRequest, "Error parsing request. %s", err)
		return
	}

	registry, err := Config.Services.Registries.RegistryFind(repo, name)
	if err != nil {
		c.String(404, "Error getting registry %q. %s", name, err)
		return
	}
	if in.Username != "" {
		registry.Username = in.Username
	}
	if in.Password != "" {
		registry.Password = in.Password
	}
	if in.Token != "" {
		registry.Token = in.Token
	}
	if in.Email != "" {
		registry.Email = in.Email
	}

	if err := registry.Validate(); err != nil {
		c.String(400, "Error updating registry. %s", err)
		return
	}
	if err := Config.Services.Registries.RegistryUpdate(repo, registry); err != nil {
		c.String(500, "Error updating registry %q. %s", in.Address, err)
		return
	}
	c.JSON(200, in.Copy())
}

// GetRegistryList gets the registry list from the database and writes
// to the response in json format.
func GetRegistryList(c *gin.Context) {
	repo := session.Repo(c)
	list, err := Config.Services.Registries.RegistryList(repo)
	if err != nil {
		c.String(500, "Error getting registry list. %s", err)
		return
	}
	// copy the registry detail to remove the sensitive
	// password and token fields.
	for i, registry := range list {
		list[i] = registry.Copy()
	}
	c.JSON(200, list)
}

// DeleteRegistry deletes the named registry from the database.
func DeleteRegistry(c *gin.Context) {
	var (
		repo = session.Repo(c)
		name = c.Param("registry")
	)
	if err := Config.Services.Registries.RegistryDelete(repo, name); err != nil {
		c.String(500, "Error deleting registry %q. %s", name, err)
		return
	}
	c.String(204, "")
}
