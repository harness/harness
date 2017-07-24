package internal

import (
	"encoding/json"
	"fmt"
)

type Commit struct {
	File *File `json:"file"`
}

type File struct {
	Data string `json:"data"`
}

func (c *Client) GetFile(globalKey, projectName, ref, path string) ([]byte, error) {
	u := fmt.Sprintf("/user/%s/project/%s/git/blob/%s/%s", globalKey, projectName, ref, path)
	resp, err := c.Get(u, nil)
	if err != nil {
		return nil, err
	}
	commit := &Commit{}
	err = json.Unmarshal(resp, commit)
	if err != nil {
		return nil, APIClientErr{"fail to parse file data", u, err}
	}
	if commit == nil || commit.File == nil {
		return nil, nil
	}
	return []byte(commit.File.Data), nil
}
