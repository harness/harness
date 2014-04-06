package github

import (
	"encoding/base64"
	"fmt"
)

// These API methods let you retrieve the contents of files within a repository
// as Base64 encoded content.
type ContentResource struct {
	client *Client
}

type Content struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Encoding string `json:"encoding"`
	Content  string `json:"content"`
	Size     int64  `json:"size"`
	Sha      string `json:"sha"`
}

func (c *Content) DecodeContent() ([]byte, error) {
	return base64.StdEncoding.DecodeString(c.Content)
}

// This method returns the contents of a file or directory in a repository.
func (r *ContentResource) Find(owner, repo, path string) (*Content, error) {
	content := Content{}
	url_path := fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, path)
	if err := r.client.do("GET", url_path, nil, &content); err != nil {
		return nil, err
	}

	return &content, nil
}

// This method returns the contents of a file or directory in a repository.
func (r *ContentResource) FindRef(owner, repo, path, ref string) (*Content, error) {
	content := Content{}
	url_path := fmt.Sprintf("/repos/%s/%s/contents/%s?ref=%s", owner, repo, path, ref)
	if err := r.client.do("GET", url_path, nil, &content); err != nil {
		return nil, err
	}

	return &content, nil
}

// This method returns the preferred README for a repository.
func (r *ContentResource) ReadMe(owner, repo string) (*Content, error) {
	content := Content{}
	path := fmt.Sprintf("/repos/%s/%s/readme", owner, repo)
	if err := r.client.do("GET", path, nil, &content); err != nil {
		return nil, err
	}

	return &content, nil
}
