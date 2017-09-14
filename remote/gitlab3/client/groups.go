package client

import (
	"encoding/json"
	"strconv"
)

const (
	groupsUrl = "/groups"
)

// Get a list of all projects owned by the authenticated user.
func (g *Client) AllGroups() ([]*Namespace, error) {
	var perPage = 100
	var groups []*Namespace

	for i := 1; true; i++ {
		contents, err := g.Groups(i, perPage)
		if err != nil {
			return groups, err
		}

		for _, value := range contents {
			groups = append(groups, value)
		}

		if len(groups) == 0 {
			break
		}

		if len(groups)/i < perPage {
			break
		}
	}

	return groups, nil
}

func (g *Client) Groups(page, perPage int) ([]*Namespace, error) {
	url, opaque := g.ResourceUrl(groupsUrl, nil, QMap{
		"page":     strconv.Itoa(page),
		"per_page": strconv.Itoa(perPage),
	})

	var groups []*Namespace

	contents, err := g.Do("GET", url, opaque, nil)
	if err == nil {
		err = json.Unmarshal(contents, &groups)
	}

	return groups, err
}
