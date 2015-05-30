package client

import (
	"net"
	"net/http"
	"net/rpc"

	"github.com/drone/drone/pkg/config"
	common "github.com/drone/drone/pkg/types"
)

// Client communicates with a Remote plugin using the
// net/rpc protocol.
type Client struct {
	*rpc.Client
}

// New returns a new, remote datastore backend that connects
// via tcp and exchanges data using Go's RPC mechanism.
func New(conf *config.Config) (*Client, error) {
	conn, err := net.Dial("tcp", conf.Server.Addr)
	if err != nil {
		return nil, err
	}
	client := &Client{
		rpc.NewClient(conn),
	}
	return client, nil
}

func (c *Client) Login(token, secret string) (*common.User, error) {
	return nil, nil
}

// Repo fetches the named repository from the remote system.
func (c *Client) Repo(u *common.User, owner, repo string) (*common.Repo, error) {
	return nil, nil
}

func (c *Client) Perm(u *common.User, owner, repo string) (*common.Perm, error) {
	return nil, nil
}

func (c *Client) Script(u *common.User, r *common.Repo, b *common.Build) ([]byte, error) {
	return nil, nil
}

func (c *Client) Status(u *common.User, r *common.Repo, b *common.Build, link string) error {
	return nil
}

func (c *Client) Activate(u *common.User, r *common.Repo, k *common.Keypair, link string) error {
	return nil
}

func (c *Client) Deactivate(u *common.User, r *common.Repo, link string) error {
	return nil
}

func (c *Client) Hook(r *http.Request) (*common.Hook, error) {
	hook := new(common.Hook)
	header := make(http.Header)
	copyHeader(r.Header, header)

	return hook, nil
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
