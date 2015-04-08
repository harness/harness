package rpc

import (
	"time"

	"github.com/drone/drone/common"
)

type GetBuildReq struct {
	Repo  string
	Build int
}

type GetBuildResp struct {
	Build *common.Build
}

func (c *Client) GetBuild(repo string, build int) (*common.Build, error) {
	req := &GetBuildReq{repo, build}
	res := &GetBuildResp{}
	err := c.Call("Datastore.GetBuild", req, res)
	return res.Build, err
}

type GetBuildListReq struct {
	Repo string
}

type GetBuildListResp struct {
	Builds []*common.Build
}

func (c *Client) GetBuildList(repo string) ([]*common.Build, error) {
	req := &GetBuildListReq{repo}
	res := &GetBuildListResp{}
	err := c.Call("Datastore.GetBuildList", req, res)
	return res.Builds, err
}

type GetBuildLastReq struct {
	Repo string
}

type GetBuildLastResp struct {
	Build *common.Build
}

func (c *Client) GetBuildLast(repo string) (*common.Build, error) {
	req := &GetBuildLastReq{repo}
	res := &GetBuildLastResp{}
	err := c.Call("Datastore.GetBuildLast", req, res)
	return res.Build, err
}

type GetBuildStatusReq struct {
	Repo   string
	Build  int
	Status string
}

type GetBuildStatusResp struct {
	Status *common.Status
}

func (c *Client) GetBuildStatus(repo string, build int, status string) (*common.Status, error) {
	req := &GetBuildStatusReq{repo, build, status}
	res := &GetBuildStatusResp{}
	err := c.Call("Datastore.GetBuildStatus", req, res)
	return res.Status, err
}

type GetBuildStatusListReq struct {
	Repo  string
	Build int
}

type GetBuildStatusListResp struct {
	Statuses []*common.Status
}

func (c *Client) GetBuildStatusList(repo string, build int) ([]*common.Status, error) {
	req := &GetBuildStatusListReq{repo, build}
	res := &GetBuildStatusListResp{}
	err := c.Call("Datastore.GetBuildStatusList", req, res)
	return res.Statuses, err
}

type InsertBuildReq struct {
	Repo  string
	Build *common.Build
}

func (c *Client) InsertBuild(repo string, build *common.Build) error {
	build.Created = time.Now().UTC().Unix()
	build.Updated = time.Now().UTC().Unix()
	// TODO need to capture the sequential build number that is generated
	req := &InsertBuildReq{repo, build}
	return c.Call("Datastore.InsertBuild", req, nil)
}

type InsertBuildStatusReq struct {
	Repo   string
	Build  int
	Status *common.Status
}

func (c *Client) InsertBuildStatus(repo string, build int, status *common.Status) error {
	req := &InsertBuildStatusReq{repo, build, status}
	return c.Call("Datastore.InsertBuildStatus", req, nil)
}

type UpdateBuildReq struct {
	Repo  string
	Build *common.Build
}

func (c *Client) UpdateBuild(repo string, build *common.Build) error {
	build.Updated = time.Now().UTC().Unix()
	req := &InsertBuildReq{repo, build}
	return c.Call("Datastore.UpdateBuild", req, nil)
}
