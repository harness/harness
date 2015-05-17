package plugin

import (
	common "github.com/drone/drone/pkg/types"
)

type GetTokenReq struct {
	Sha string
}

type GetTokenResp struct {
	Token *common.Token
}

func (c *Client) GetToken(sha string) (*common.Token, error) {
	req := &GetTokenReq{sha}
	res := &GetTokenResp{}
	err := c.Call("Datastore.GetToken", req, res)
	return res.Token, err
}

type InsertTokenReq struct {
	Token *common.Token
}

func (c *Client) InsertToken(token *common.Token) error {
	req := &InsertTokenReq{token}
	return c.Call("Datastore.InsertToken", req, nil)
}

type DeleteTokenReq struct {
	Token *common.Token
}

func (c *Client) DeleteToken(token *common.Token) error {
	req := &DeleteTokenReq{token}
	return c.Call("Datastore.DeleteToken", req, nil)
}
