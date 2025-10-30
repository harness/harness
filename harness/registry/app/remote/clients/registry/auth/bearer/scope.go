// Source: https://github.com/goharbor/harbor

// Copyright 2016 Project Harbor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bearer

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/distribution/reference"
)

const (
	scopeTypeRegistry   = "registry"
	scopeTypeRepository = "repository"
	scopeActionPull     = "pull"
	scopeActionPush     = "push"
	scopeActionAll      = "*"
)

const v2 = "/v2/("

var (
	catalog  = regexp.MustCompile("/v2/_catalog$")
	tag      = regexp.MustCompile(v2 + reference.NameRegexp.String() + ")/tags/list")
	manifest = regexp.MustCompile(
		v2 +
			reference.NameRegexp.String() +
			")/manifests/(" +
			reference.TagRegexp.String() +
			"|" + reference.DigestRegexp.String() + ")",
	)
	blob = regexp.MustCompile(
		v2 + reference.NameRegexp.String() + ")/blobs/" + reference.DigestRegexp.String(),
	)
	blobUpload = regexp.MustCompile(v2 + reference.NameRegexp.String() + ")/blobs/uploads")
)

type scope struct {
	Type    string
	Name    string
	Actions []string
}

func (s *scope) String() string {
	return fmt.Sprintf("%s:%s:%s", s.Type, s.Name, strings.Join(s.Actions, ","))
}

func parseScopes(req *http.Request) []*scope {
	path := strings.TrimRight(req.URL.Path, "/")
	var scopes []*scope
	repository := ""
	// manifest
	if subs := manifest.FindStringSubmatch(path); len(subs) >= 2 {
		// manifest
		repository = subs[1]
	} else if subs1 := blob.FindStringSubmatch(path); len(subs1) >= 2 {
		// blob
		repository = subs1[1]
	} else if subs2 := blobUpload.FindStringSubmatch(path); len(subs2) >= 2 {
		// blob upload
		repository = subs2[1]
		// blob mount
		from := req.URL.Query().Get("from")
		if len(from) > 0 {
			scopes = append(
				scopes, &scope{
					Type:    scopeTypeRepository,
					Name:    from,
					Actions: []string{scopeActionPull},
				},
			)
		}
	} else if subs3 := tag.FindStringSubmatch(path); len(subs3) >= 2 {
		// tag
		repository = subs3[1]
	}
	if len(repository) > 0 {
		scp := &scope{
			Type: scopeTypeRepository,
			Name: repository,
		}
		switch req.Method {
		case http.MethodGet, http.MethodHead:
			scp.Actions = []string{scopeActionPull}
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			scp.Actions = []string{scopeActionPull, scopeActionPush}
		case http.MethodDelete:
			scp.Actions = []string{scopeActionAll}
		}
		scopes = append(scopes, scp)
		return scopes
	}

	// catalog
	if catalog.MatchString(path) {
		return []*scope{
			{
				Type:    scopeTypeRegistry,
				Name:    "catalog",
				Actions: []string{scopeActionAll},
			},
		}
	}

	// base or no match, return nil
	return nil
}
