package server

import (
	"github.com/drone/drone/Godeps/_workspace/src/github.com/gin-gonic/gin"
	"github.com/drone/drone/Godeps/_workspace/src/github.com/gin-gonic/gin/binding"
	"github.com/drone/drone/Godeps/_workspace/src/github.com/ungerik/go-gravatar"
	log "github.com/drone/drone/Godeps/_workspace/src/github.com/Sirupsen/logrus"

	"github.com/drone/drone/pkg/types"
)

// GetUsers accepts a request to retrieve all users
// from the datastore and return encoded in JSON format.
//
//     GET /api/users
//
func GetUsers(c *gin.Context) {
	store := ToDatastore(c)
	users, err := store.UserList()
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
	store := ToDatastore(c)
	name := c.Params.ByName("name")
	user := &types.User{Login: name}
	user.Token = c.Request.FormValue("token")
	user.Secret = c.Request.FormValue("secret")
	user.Hash = c.Request.FormValue("hash")
	if len(user.Hash) == 0 {
		user.Hash = types.GenerateToken()
	}
	if err := store.AddUser(user); err != nil {
		c.Fail(400, err)
	} else {
		tokenstr, err := generateUserToken(user)
		if err != nil {
			log.Errorf("cannot create token for %s. %s", user.Login, err)
			tokenstr = ""
		}
		userWithToken := struct {
			*types.User
			Token string `json:"token,omitempty"`
		}{user, tokenstr}
		c.JSON(201, userWithToken)
	}
}

// GetUser accepts a request to retrieve a user by hostname
// and login from the datastore and return encoded in JSON
// format.
//
//     GET /api/users/:name
//
func GetUser(c *gin.Context) {
	store := ToDatastore(c)
	name := c.Params.ByName("name")
	user, err := store.UserLogin(name)
	if err != nil {
		c.Fail(404, err)
	} else {
		tokenstr, err := generateUserToken(user)
		if err != nil {
			log.Errorf("cannot create token for %s. %s", user.Login, err)
			tokenstr = ""
		}
		userWithToken := struct {
			*types.User
			Token string `json:"token,omitempty"`
		}{user, tokenstr}
		c.JSON(200, userWithToken)
	}
}

// PutUser accepts a request to update an existing user in
// the system. The modified user account is returned in JSON
// format if successful.
//
//     PUT /api/users/:name
//
func PutUser(c *gin.Context) {
	store := ToDatastore(c)
	me := ToUser(c)
	name := c.Params.ByName("name")
	user, err := store.UserLogin(name)
	if err != nil {
		c.Fail(404, err)
		return
	}

	in := &types.User{}
	if !c.BindWith(in, binding.JSON) {
		return
	}
	user.Email = in.Email
	user.Avatar = gravatar.Hash(user.Email)

	// an administrator must not be able to
	// downgrade her own account.
	if me.Login != user.Login {
		user.Admin = in.Admin
	}

	err = store.SetUser(user)
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
	store := ToDatastore(c)
	me := ToUser(c)
	name := c.Params.ByName("name")
	user, err := store.UserLogin(name)
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

	if err := store.DelUser(user); err != nil {
		c.Fail(400, err)
	} else {
		c.Writer.WriteHeader(204)
	}
}
