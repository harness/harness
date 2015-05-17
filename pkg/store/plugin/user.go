package plugin

import (
	"time"

	common "github.com/drone/drone/pkg/types"
)

type GetUserReq struct {
	Login string
}

type GetUserResp struct {
	User *common.User
}

func (c *Client) GetUser(login string) (*common.User, error) {
	req := &GetUserReq{login}
	res := &GetUserResp{}
	err := c.Call("Datastore.GetUser", req, res)
	return res.User, err
}

type GetUserTokensReq struct {
	Login string
}

type GetUserTokensResp struct {
	Tokens []*common.Token
}

func (c *Client) GetUserTokens(login string) ([]*common.Token, error) {
	req := &GetUserTokensReq{login}
	res := &GetUserTokensResp{}
	err := c.Call("Datastore.GetUserTokens", req, res)
	return res.Tokens, err
}

type GetUserReposReq struct {
	Login string
}

type GetUserReposResp struct {
	Repos []*common.Repo
}

func (c *Client) GetUserRepos(login string) ([]*common.Repo, error) {
	req := &GetUserReposReq{login}
	res := &GetUserReposResp{}
	err := c.Call("Datastore.GetUserRepos", req, res)
	return res.Repos, err
}

type GetUserCountResp struct {
	Count int
}

func (c *Client) GetUserCount() (int, error) {
	res := &GetUserCountResp{}
	err := c.Call("Datastore.GetUserCount", nil, res)
	return res.Count, err
}

type GetUserListResp struct {
	Users []*common.User
}

func (c *Client) GetUserList() ([]*common.User, error) {
	res := &GetUserListResp{}
	err := c.Call("Datastore.GetUserList", nil, res)
	return res.Users, err
}

type UpdateUserReq struct {
	User *common.User
}

func (c *Client) UpdateUser(user *common.User) error {
	user.Updated = time.Now().UTC().Unix()
	req := &UpdateUserReq{user}
	return c.Call("Datastore.UpdateUser", req, nil)
}

type InsertUserReq struct {
	User *common.User
}

func (c *Client) InsertUser(user *common.User) error {
	user.Created = time.Now().UTC().Unix()
	user.Updated = time.Now().UTC().Unix()
	req := &InsertUserReq{user}
	return c.Call("Datastore.InsertUser", req, nil)
}

type DeleteUserReq struct {
	User *common.User
}

func (c *Client) DeleteUser(user *common.User) error {
	req := &DeleteUserReq{user}
	return c.Call("Datastore.DeleteUser", req, nil)
}
