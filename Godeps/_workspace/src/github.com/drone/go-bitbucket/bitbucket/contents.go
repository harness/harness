package bitbucket

import (
	"fmt"
)

type Source struct {
	Node string `json:"node"`
	Path string `json:"path"`
	Data string `json:"data"`
	Size int64  `json:"size"`
}

// Use the Bitbucket src resource to browse directories and view files.
// This is a read-only resource.
//
// https://confluence.atlassian.com/display/BITBUCKET/src+Resources
type SourceResource struct {
	client *Client
}

// Gets information about an individual file in a repository
func (r *SourceResource) Find(owner, slug, revision, path string) (*Source, error) {
	src := Source{}
	url_path := fmt.Sprintf("/repositories/%s/%s/src/%s/%s", owner, slug, revision, path)

	if err := r.client.do("GET", url_path, nil, nil, &src); err != nil {
		return nil, err
	}

	return &src, nil
}

// Gets a list of the src in a repository.
func (r *SourceResource) List(owner, slug, revision, path string) ([]*Source, error) {
	src := []*Source{}
	url_path := fmt.Sprintf("/repositories/%s/%s/src/%s/%s", owner, slug, revision, path)
	if err := r.client.do("GET", url_path, nil, nil, &src); err != nil {
		return nil, err
	}

	return src, nil
}
