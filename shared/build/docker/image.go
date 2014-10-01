package docker

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/parsers"
)

type Images struct {
	ID          string   `json:"Id"`
	RepoTags    []string `json:",omitempty"`
	Created     int64
	Size        int64
	VirtualSize int64
	ParentId    string `json:",omitempty"`

	// DEPRECATED
	Repository string `json:",omitempty"`
	Tag        string `json:",omitempty"`
}

type Image struct {
	ID              string    `json:"id"`
	Parent          string    `json:"parent,omitempty"`
	Comment         string    `json:"comment,omitempty"`
	Created         time.Time `json:"created"`
	Container       string    `json:"container,omitempty"`
	ContainerConfig Config    `json:"container_config,omitempty"`
	DockerVersion   string    `json:"docker_version,omitempty"`
	Author          string    `json:"author,omitempty"`
	Config          *Config   `json:"config,omitempty"`
	Architecture    string    `json:"architecture,omitempty"`
	OS              string    `json:"os,omitempty"`
	Size            int64
}

type Delete struct {
	Deleted  string `json:",omitempty"`
	Untagged string `json:",omitempty"`
}

type ImageService struct {
	*Client
}

// List Images
func (c *ImageService) List() ([]*Images, error) {
	images := []*Images{}
	err := c.do("GET", "/images/json?all=0", nil, &images)
	return images, err
}

// Create an image, either by pull it from the registry or by importing it.
func (c *ImageService) Create(image string) error {
	return c.do("POST", fmt.Sprintf("/images/create?fromImage=%s", image), nil, nil)
}

func (c *ImageService) Pull(image string) error {
	name, tag := parsers.ParseRepositoryTag(image)
	if len(tag) == 0 {
		tag = DEFAULTTAG
	}
	return c.PullTag(name, tag)
}

func (c *ImageService) PullTag(name, tag string) error {
	var out io.Writer
	if Logging {
		out = os.Stdout
	}

	path := fmt.Sprintf("/images/create?fromImage=%s&tag=%s", name, tag)
	return c.stream("POST", path, nil, out, http.Header{})
}

// Remove the image name from the filesystem
func (c *ImageService) Remove(image string) ([]*Delete, error) {
	resp := []*Delete{}
	err := c.do("DELETE", fmt.Sprintf("/images/%s", image), nil, &resp)
	return resp, err
}

// Inspect the image
func (c *ImageService) Inspect(name string) (*Image, error) {
	image := Image{}
	err := c.do("GET", fmt.Sprintf("/images/%s/json", name), nil, &image)
	return &image, err
}

// Build the Image
func (c *ImageService) Build(tag, dir string) error {

	// tar the file
	context, err := archive.Tar(dir, archive.Uncompressed)
	if err != nil {
		return err
	}

	var body io.Reader
	body = ioutil.NopCloser(context)

	// Upload the build context
	v := url.Values{}
	v.Set("t", tag)
	v.Set("q", "1")
	v.Set("rm", "1")

	// url path
	path := fmt.Sprintf("/build?%s", v.Encode())

	// set content type to tar file
	headers := http.Header{}
	headers.Set("Content-Type", "application/tar")

	// make the request
	return c.stream("POST", path, body /*os.Stdout*/, nil, headers)
}
