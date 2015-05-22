package server

import (
	"fmt"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/gin-gonic/gin"
	"github.com/drone/drone/Godeps/_workspace/src/github.com/gin-gonic/gin/binding"

	"github.com/drone/drone/pkg/remote"
	common "github.com/drone/drone/pkg/types"
	"github.com/drone/drone/pkg/utils/httputil"
	"github.com/drone/drone/pkg/utils/sshutil"
)

// repoResp is a data structure used for sending
// repository data to the client, augmented with
// additional repository meta-data.
type repoResp struct {
	*common.Repo
	Perms   *common.Perm      `json:"permissions,omitempty"`
	Keypair *common.Keypair   `json:"keypair,omitempty"`
	Params  map[string]string `json:"params,omitempty"`
	Starred bool              `json:"starred,omitempty"`
}

// repoReq is a data structure used for receiving
// repository data from the client to modify the
// attributes of an existing repository.
//
// note that attributes are pointers so that we can
// accept null values, effectively patching an existing
// repository object with only the supplied fields.
type repoReq struct {
	PostCommit  *bool  `json:"post_commits"`
	PullRequest *bool  `json:"pull_requests"`
	Trusted     *bool  `json:"privileged"`
	Timeout     *int64 `json:"timeout"`

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
	data := repoResp{repo, perm, nil, nil, false}

	// if the user is authenticated, we should display
	// if she is watching the current repository.
	if user == nil {
		c.JSON(200, data)
		return
	}

	// if the user is an administrator of the project
	// we should display the private parameter data
	// and keypair data.
	if perm.Push {
		data.Params = repo.Params
		data.Keypair = &common.Keypair{
			Public: repo.PublicKey,
		}
	}
	// check to see if the user is subscribing to the repo
	data.Starred, _ = store.Starred(user, repo)

	c.JSON(200, data)
}

// PutRepo accepts a request to update the named repository
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
		repo.Params = *in.Params
	}

	if in.PostCommit != nil {
		repo.PullRequest = *in.PullRequest
	}
	if in.PullRequest != nil {
		repo.PullRequest = *in.PullRequest
	}
	if in.Trusted != nil && user.Admin {
		repo.Trusted = *in.Trusted
	}
	if in.Timeout != nil && user.Admin {
		repo.Timeout = *in.Timeout
	}

	err := store.SetRepo(repo)
	if err != nil {
		c.Fail(400, err)
		return
	}

	data := repoResp{repo, perm, nil, nil, false}
	data.Params = repo.Params
	data.Keypair = &common.Keypair{
		Public: repo.PublicKey,
	}
	data.Starred, _ = store.Starred(user, repo)

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

	err = ds.DelRepo(r)
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
	sess := ToSession(c)
	store := ToDatastore(c)
	owner := c.Params.ByName("owner")
	name := c.Params.ByName("name")

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

	// error if the repository already exists
	_, err = store.RepoName(owner, name)
	if err == nil {
		c.String(409, "Repository already exists")
		return
	}

	token := &common.Token{}
	token.Kind = common.TokenHook
	token.Label = r.FullName
	tokenstr, err := sess.GenerateToken(token)
	if err != nil {
		c.Fail(500, err)
		return
	}

	link := fmt.Sprintf(
		"%s/api/hook?access_token=%s",
		httputil.GetURL(c.Request),
		tokenstr,
	)

	// set the repository owner to the
	// currently authenticated user.
	r.UserID = user.ID
	r.PostCommit = true
	r.PullRequest = true
	r.Timeout = 60 // 1 hour default build time
	r.Self = fmt.Sprintf(
		"%s/%s",
		httputil.GetURL(c.Request),
		r.FullName,
	)

	// generate an RSA key and add to the repo
	key, err := sshutil.GeneratePrivateKey()
	if err != nil {
		c.Fail(400, err)
		return
	}
	r.PublicKey = string(sshutil.MarshalPublicKey(&key.PublicKey))
	r.PrivateKey = string(sshutil.MarshalPrivateKey(key))
	keypair := &common.Keypair{
		Public:  r.PublicKey,
		Private: r.PrivateKey,
	}

	// activate the repository before we make any
	// local changes to the database.
	err = remote.Activate(user, r, keypair, link)
	if err != nil {
		c.Fail(500, err)
		return
	}

	// persist the repository
	err = store.AddRepo(r)
	if err != nil {
		c.Fail(500, err)
		return
	}

	store.AddStar(user, r)

	c.JSON(200, r)
}

// Unubscribe accapets a request to unsubscribe the
// currently authenticated user to the repository.
//
//     DEL /api/subscribers/:owner/:name
//
func Unsubscribe(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	user := ToUser(c)

	err := store.DelStar(user, repo)
	if err != nil {
		c.Fail(400, err)
	} else {
		c.Writer.WriteHeader(200)
	}
}

// Subscribe accapets a request to subscribe the
// currently authenticated user to the repository.
//
//     POST /api/subscriber/:owner/:name
//
func Subscribe(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	user := ToUser(c)

	err := store.AddStar(user, repo)
	if err != nil {
		c.Fail(400, err)
	} else {
		c.Writer.WriteHeader(200)
	}
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
