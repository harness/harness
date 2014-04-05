package gogitlab

import (
	"encoding/json"
	"net/url"
)

const (
	// ID
	project_url_deploy_keys = "/projects/:id/keys" // Get list of project deploy keys
	// PROJECT ID AND KEY ID
	project_url_deploy_key = "/projects/:id/keys/:key_id" // Get single project deploy key
)

/*
Get list of project deploy keys.

    GET /projects/:id/keys

Parameters:

    id The ID of a project

*/
func (g *Gitlab) ProjectDeployKeys(id string) ([]*PublicKey, error) {

	url, opaque := g.ResourceUrlRaw(project_url_deploy_keys, map[string]string{":id": id})

	var deployKeys []*PublicKey

	contents, err := g.buildAndExecRequestRaw("GET", url, opaque, nil)
	if err == nil {
		err = json.Unmarshal(contents, &deployKeys)
	}

	return deployKeys, err
}

/*
Get single project deploy key.

    GET /projects/:id/keys/:key_id

Parameters:

    id     The ID of a project
    key_id The ID of a key

*/
func (g *Gitlab) ProjectDeployKey(id, key_id string) (*PublicKey, error) {

	url, opaque := g.ResourceUrlRaw(project_url_deploy_key, map[string]string{
		":id":     id,
		":key_id": key_id,
	})

	var deployKey *PublicKey

	contents, err := g.buildAndExecRequestRaw("GET", url, opaque, nil)
	if err == nil {
		err = json.Unmarshal(contents, &deployKey)
	}

	return deployKey, err
}

/*
Add deploy key to project.

    POST /projects/:id/keys

Parameters:

    id    The ID of a project
    title The key title
    key   The key value

*/
func (g *Gitlab) AddProjectDeployKey(id, title, key string) error {

	path, opaque := g.ResourceUrlRaw(project_url_deploy_keys, map[string]string{":id": id})

	var err error

	v := url.Values{}
	v.Set("title", title)
	v.Set("key", key)

	body := v.Encode()

	_, err = g.buildAndExecRequestRaw("POST", path, opaque, []byte(body))

	return err
}

/*
Remove deploy key from project

    DELETE /projects/:id/keys/:key_id

Parameters:

    id     The ID of a project
    key_id The ID of a key

*/
func (g *Gitlab) RemoveProjectDeployKey(id, key_id string) error {

	url, opaque := g.ResourceUrlRaw(project_url_deploy_key, map[string]string{
		":id":     id,
		":key_id": key_id,
	})

	var err error

	_, err = g.buildAndExecRequestRaw("DELETE", url, opaque, nil)

	return err
}
