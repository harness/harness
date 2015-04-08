package server

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/drone/drone/common"
	"github.com/drone/drone/common/gravatar"
)

// GetUsers accepts a request to retrieve all users
// from the datastore and return encoded in JSON format.
//
//     GET /api/users
//
func GetUsers(c *gin.Context) {
	ds := ToDatastore(c)
	users, err := ds.GetUserList()
	if err != nil {
		c.Fail(400, err)
	} else {
		c.JSON(200, users)
	}
}

// PostUser accepts a request to create a new user in the
// system. The created user account is returned in JSON
// format if successful.
//
//     POST /api/users
//
func PostUser(c *gin.Context) {
	ds := ToDatastore(c)
	name := c.Params.ByName("name")
	user := &common.User{Login: name, Name: name}
	user.Token = c.Request.FormValue("token")
	user.Secret = c.Request.FormValue("secret")
	if err := ds.InsertUser(user); err != nil {
		c.Fail(400, err)
	} else {
		c.JSON(201, user)
	}
}

// GetUser accepts a request to retrieve a user by hostname
// and login from the datastore and return encoded in JSON
// format.
//
//     GET /api/users/:name
//
func GetUser(c *gin.Context) {
	ds := ToDatastore(c)
	name := c.Params.ByName("name")
	user, err := ds.GetUser(name)
	if err != nil {
		c.Fail(404, err)
	} else {
		c.JSON(200, user)
	}
}

// PutUser accepts a request to update an existing user in
// the system. The modified user account is returned in JSON
// format if successful.
//
//     PUT /api/users/:name
//
func PutUser(c *gin.Context) {
	ds := ToDatastore(c)
	me := ToUser(c)
	name := c.Params.ByName("name")
	user, err := ds.GetUser(name)
	if err != nil {
		c.Fail(404, err)
		return
	}

	in := &common.User{}
	if !c.BindWith(in, binding.JSON) {
		return
	}
	user.Email = in.Email
	user.Gravatar = gravatar.Generate(user.Email)

	// an administrator must not be able to
	// downgrade her own account.
	if me.Login != user.Login {
		user.Admin = in.Admin
	}

	err = ds.UpdateUser(user)
	if err != nil {
		c.Fail(400, err)
	} else {
		c.JSON(200, user)
	}
}

// DeleteUser accepts a request to delete the specified
// user account from the system. A successful request will
// respond with an OK 200 status.
//
//     DELETE /api/users/:name
//
func DeleteUser(c *gin.Context) {
	ds := ToDatastore(c)
	me := ToUser(c)
	name := c.Params.ByName("name")
	user, err := ds.GetUser(name)
	if err != nil {
		c.Fail(404, err)
		return
	}

	// an administrator must not be able to
	// delete her own account.
	if user.Login == me.Login {
		c.Writer.WriteHeader(403)
		return
	}

	if err := ds.DeleteUser(user); err != nil {
		c.Fail(400, err)
	} else {
		c.Writer.WriteHeader(204)
	}
}
