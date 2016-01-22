// Copyright 2015 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package gogs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type CreateOrgOption struct {
	UserName    string `json:"username" binding:"Required"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	Website     string `json:"website"`
	Location    string `json:"location"`
}

func (c *Client) AdminCreateOrg(user string, opt CreateOrgOption) (*Organization, error) {
	body, err := json.Marshal(&opt)
	if err != nil {
		return nil, err
	}
	org := new(Organization)
	return org, c.getParsedResponse("POST", fmt.Sprintf("/admin/users/%s/orgs", user),
		http.Header{"content-type": []string{"application/json"}}, bytes.NewReader(body), org)
}
