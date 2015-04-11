package server

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/drone/drone/common"
	"github.com/drone/drone/common/httputil"
	"github.com/drone/drone/common/sshutil"
	"github.com/drone/drone/remote"
)

// repoResp is a data structure used for sending
// repository data to the client, augmented with
// additional repository meta-data.
type repoResp struct {
	*common.Repo
	Perms  *common.Perm       `json:"permissions,omitempty"`
	Watch  *common.Subscriber `json:"subscription,omitempty"`
	Params map[string]string  `json:"params,omitempty"`
}

// repoReq is a data structure used for receiving
// repository data from the client to modify the
// attributes of an existing repository.
//
// note that attributes are pointers so that we can
// accept null values, effectively patching an existing
// repository object with only the supplied fields.
type repoReq struct {
	Disabled   *bool  `json:"disabled"`
	DisablePR  *bool  `json:"disable_prs"`
	DisableTag *bool  `json:"disable_tags"`
	Trusted    *bool  `json:"privileged"`
	Timeout    *int64 `json:"timeout"`

	// optional private parameters can only be
	// supplied by the repository admin.
	Params *map[string]string `json:"params"`
}

// GetRepo accepts a request to retrieve a commit
// from the datastore for the given repository, branch and
// commit hash.
//
//     GET /api/repos/:owner/:name
//
func GetRepo(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	user := ToUser(c)
	perm := ToPerm(c)
	data := repoResp{repo, perm, nil, nil}
	// if the user is an administrator of the project
	// we should display the private parameter data.
	if perm.Admin {
		data.Params, _ = store.GetRepoParams(repo.FullName)
	}
	// if the user is authenticated, we should display
	// if she is watching the current repository.
	if user == nil {
		c.JSON(200, data)
		return
	}

	// check to see if the user is subscribing to the repo
	_, ok := user.Repos[repo.FullName]
	data.Watch = &common.Subscriber{Subscribed: ok}

	c.JSON(200, data)
}

// PutRepo accapets a request to update the named repository
// in the datastore. It expects a JSON input and returns the
// updated repository in JSON format if successful.
//
//     PUT /api/repos/:owner/:name
//
func PutRepo(c *gin.Context) {
	store := ToDatastore(c)
	perm := ToPerm(c)
	user := ToUser(c)
	repo := ToRepo(c)

	in := &repoReq{}
	if !c.BindWith(in, binding.JSON) {
		return
	}

	if in.Params != nil {
		err := store.UpsertRepoParams(repo.FullName, *in.Params)
		if err != nil {
			c.Fail(400, err)
			return
		}
	}
	if in.Disabled != nil {
		repo.Disabled = *in.Disabled
	}
	if in.DisablePR != nil {
		repo.DisablePR = *in.DisablePR
	}
	if in.DisableTag != nil {
		repo.DisableTag = *in.DisableTag
	}
	if in.Trusted != nil && user.Admin {
		repo.Trusted = *in.Trusted
	}
	if in.Timeout != nil && user.Admin {
		repo.Timeout = *in.Timeout
	}

	err := store.UpdateRepo(repo)
	if err != nil {
		c.Fail(400, err)
		return
	}

	data := repoResp{repo, perm, nil, nil}
	data.Params, _ = store.GetRepoParams(repo.FullName)

	// check to see if the user is subscribing to the repo
	_, ok := user.Repos[repo.FullName]
	data.Watch = &common.Subscriber{Subscribed: ok}

	c.JSON(200, data)
}

// DeleteRepo accepts a request to delete the named
// repository.
//
//     DEL /api/repos/:owner/:name
//
func DeleteRepo(c *gin.Context) {
	ds := ToDatastore(c)
	u := ToUser(c)
	r := ToRepo(c)

	link := fmt.Sprintf(
		"%s/api/hook",
		httputil.GetURL(c.Request),
	)

	remote := ToRemote(c)
	err := remote.Deactivate(u, r, link)
	if err != nil {
		c.Fail(400, err)
	}

	err = ds.DeleteRepo(r)
	if err != nil {
		c.Fail(400, err)
	}
	c.Writer.WriteHeader(200)
}

// PostRepo accapets a request to activate the named repository
// in the datastore. It returns a 201 status created if successful
//
//     POST /api/repos/:owner/:name
//
func PostRepo(c *gin.Context) {
	user := ToUser(c)
	store := ToDatastore(c)
	owner := c.Params.ByName("owner")
	name := c.Params.ByName("name")

	link := fmt.Sprintf(
		"%s/api/hook",
		httputil.GetURL(c.Request),
	)

	// TODO(bradrydzewski) verify repo not exists

	// get the repository and user permissions
	// from the remote system.
	remote := ToRemote(c)
	r, err := remote.Repo(user, owner, name)
	if err != nil {
		c.Fail(400, err)
	}
	m, err := remote.Perm(user, owner, name)
	if err != nil {
		c.Fail(400, err)
		return
	}
	if !m.Admin {
		c.Fail(403, fmt.Errorf("must be repository admin"))
		return
	}

	// set the repository owner to the
	// currently authenticated user.
	r.User = &common.Owner{Login: user.Login}

	// generate an RSA key and add to the repo
	key, err := sshutil.GeneratePrivateKey()
	if err != nil {
		c.Fail(400, err)
		return
	}
	keypair := &common.Keypair{}
	keypair.Public = sshutil.MarshalPublicKey(&key.PublicKey)
	keypair.Private = sshutil.MarshalPrivateKey(key)

	// activate the repository before we make any
	// local changes to the database.
	err = remote.Activate(user, r, keypair, link)
	if err != nil {
		c.Fail(500, err)
		return
	}
	println(link)

	// persist the repository
	err = store.InsertRepo(user, r)
	if err != nil {
		c.Fail(500, err)
		return
	}

	// persisty the repository key pair
	err = store.UpsertRepoKeys(r.FullName, keypair)
	if err != nil {
		c.Fail(500, err)
		return
	}

	// subscribe the user to the repository
	// if this fails we'll ignore, since the user
	// can just go click the "watch" button in the
	// user interface.
	if user.Repos == nil {
		user.Repos = map[string]struct{}{}
	}
	user.Repos[r.FullName] = struct{}{}
	store.UpdateUser(user)

	c.JSON(200, r)
}

// perms is a helper function that returns user permissions
// for a particular repository.
func perms(remote remote.Remote, u *common.User, r *common.Repo) *common.Perm {
	switch {
	case u == nil && r.Private:
		return &common.Perm{}
	case u == nil && r.Private == false:
		return &common.Perm{Pull: true}
	case u.Admin:
		return &common.Perm{Pull: true, Push: true, Admin: true}
	}

	p, err := remote.Perm(u, r.Owner, r.Name)
	if err != nil {
		return &common.Perm{}
	}
	return p
}
