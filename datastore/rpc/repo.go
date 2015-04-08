package rpc

import (
	"time"

	"github.com/drone/drone/common"
)

type GetRepoReq struct {
	Repo string
}

type GetRepoResp struct {
	Repo *common.Repo
}

func (c *Client) GetRepo(repo string) (*common.Repo, error) {
	req := &GetRepoReq{repo}
	res := &GetRepoResp{}
	err := c.Call("Datastore.GetRepo", req, res)
	return res.Repo, err
}

type GetRepoParamsReq struct {
	Repo string
}

type GetRepoParamsResp struct {
	Params map[string]string
}

func (c *Client) GetRepoParams(repo string) (map[string]string, error) {
	req := &GetRepoParamsReq{repo}
	res := &GetRepoParamsResp{}
	err := c.Call("Datastore.GetRepoParams", req, res)
	return res.Params, err
}

type GetRepoKeysReq struct {
	Repo string
}

type GetRepoKeysResp struct {
	Keys *common.Keypair
}

func (c *Client) GetRepoKeys(repo string) (*common.Keypair, error) {
	req := &GetRepoKeysReq{repo}
	res := &GetRepoKeysResp{}
	err := c.Call("Datastore.GetRepoKeys", req, res)
	return res.Keys, err
}

type UpdateRepoReq struct {
	Repo *common.Repo
}

func (c *Client) UpdateRepo(repo *common.Repo) error {
	repo.Updated = time.Now().UTC().Unix()
	req := &UpdateRepoReq{repo}
	return c.Call("Datastore.UpdateRepo", req, nil)
}

type InsertRepoReq struct {
	User *common.User
	Repo *common.Repo
}

func (c *Client) InsertRepo(user *common.User, repo *common.Repo) error {
	repo.Created = time.Now().UTC().Unix()
	repo.Updated = time.Now().UTC().Unix()
	req := &InsertRepoReq{user, repo}
	return c.Call("Datastore.InsertRepo", req, nil)
}

type UpsertRepoParamsReq struct {
	Repo string
}

func (c *Client) UpsertRepoParams(repo string, params map[string]string) error {
	req := &UpsertRepoParamsReq{repo}
	return c.Call("Datastore.UpsertRepoParams", req, nil)
}

type UpsertRepoKeysReq struct {
	Repo string
	Keys *common.Keypair
}

func (c *Client) UpsertRepoKeys(repo string, keypair *common.Keypair) error {
	req := &UpsertRepoKeysReq{repo, keypair}
	return c.Call("Datastore.UpsertRepoKeys", req, nil)
}

type DeleteRepoReq struct {
	Repo *common.Repo
}

func (c *Client) DeleteRepo(repo *common.Repo) error {
	req := &DeleteRepoReq{repo}
	return c.Call("Datastore.DeleteRepo", req, nil)
}
