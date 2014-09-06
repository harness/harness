package testdata

import (
	"net/http"
	"net/http/httptest"
)

// setup a mock server for testing purposes.
func NewServer() *httptest.Server {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	// handle requests and serve mock data
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		//println(r.URL.Path + "  " + r.Method)
		// evaluate the path to serve a dummy data file
		switch r.URL.Path {
		case "/api/v3/projects":
			w.Write(projectsPayload)
			return
		case "/api/v3/projects/4":
			w.Write(project4Paylod)
			return
		case "/api/v3/projects/6":
			w.Write(project6Paylod)
			return
		case "/api/v3/session":
			w.Write(sessionPayload)
			return
		}

		// else return a 404
		http.NotFound(w, r)
	})

	// return the server to the client which
	// will need to know the base URL path
	return server
}

// sample repository list
var projectsPayload = []byte(`
[
	{
		"id": 4,
		"description": null,
		"default_branch": "master",
		"public": false,
		"visibility_level": 0,
		"ssh_url_to_repo": "git@example.com:diaspora/diaspora-client.git",
		"http_url_to_repo": "http://example.com/diaspora/diaspora-client.git",
		"web_url": "http://example.com/diaspora/diaspora-client",
		"owner": {
			"id": 3,
			"name": "Diaspora",
			"created_at": "2013-09-30T13: 46: 02Z"
		},
		"name": "Diaspora Client",
		"name_with_namespace": "Diaspora / Diaspora Client",
		"path": "diaspora-client",
		"path_with_namespace": "diaspora/diaspora-client",
		"issues_enabled": true,
		"merge_requests_enabled": true,
		"wiki_enabled": true,
		"snippets_enabled": false,
		"created_at": "2013-09-30T13: 46: 02Z",
		"last_activity_at": "2013-09-30T13: 46: 02Z",
		"namespace": {
			"created_at": "2013-09-30T13: 46: 02Z",
			"description": "",
			"id": 3,
			"name": "Diaspora",
			"owner_id": 1,
			"path": "diaspora",
			"updated_at": "2013-09-30T13: 46: 02Z"
		},
		"archived": false
	},
	{
		"id": 6,
		"description": null,
		"default_branch": "master",
		"public": false,
		"visibility_level": 0,
		"ssh_url_to_repo": "git@example.com:brightbox/puppet.git",
		"http_url_to_repo": "http://example.com/brightbox/puppet.git",
		"web_url": "http://example.com/brightbox/puppet",
		"owner": {
			"id": 4,
			"name": "Brightbox",
			"created_at": "2013-09-30T13:46:02Z"
		},
		"name": "Puppet",
		"name_with_namespace": "Brightbox / Puppet",
		"path": "puppet",
		"path_with_namespace": "brightbox/puppet",
		"issues_enabled": true,
		"merge_requests_enabled": true,
		"wiki_enabled": true,
		"snippets_enabled": false,
		"created_at": "2013-09-30T13:46:02Z",
		"last_activity_at": "2013-09-30T13:46:02Z",
		"namespace": {
			"created_at": "2013-09-30T13:46:02Z",
			"description": "",
			"id": 4,
			"name": "Brightbox",
			"owner_id": 1,
			"path": "brightbox",
			"updated_at": "2013-09-30T13:46:02Z"
		},
		"archived": false
	}
]
`)

var project4Paylod = []byte(`
{
	"id": 4,
	"description": null,
	"default_branch": "master",
	"public": false,
	"visibility_level": 0,
	"ssh_url_to_repo": "git@example.com:diaspora/diaspora-client.git",
	"http_url_to_repo": "http://example.com/diaspora/diaspora-client.git",
	"web_url": "http://example.com/diaspora/diaspora-client",
	"owner": {
		"id": 3,
		"name": "Diaspora",
		"created_at": "2013-09-30T13: 46: 02Z"
	},
	"name": "Diaspora Client",
	"name_with_namespace": "Diaspora / Diaspora Client",
	"path": "diaspora-client",
	"path_with_namespace": "diaspora/diaspora-client",
	"issues_enabled": true,
	"merge_requests_enabled": true,
	"wiki_enabled": true,
	"snippets_enabled": false,
	"created_at": "2013-09-30T13: 46: 02Z",
	"last_activity_at": "2013-09-30T13: 46: 02Z",
	"namespace": {
		"created_at": "2013-09-30T13: 46: 02Z",
		"description": "",
		"id": 3,
		"name": "Diaspora",
		"owner_id": 1,
		"path": "diaspora",
		"updated_at": "2013-09-30T13: 46: 02Z"
	},
	"archived": false,
	"permissions": {
		"project_access": {
			"access_level": 10,
			"notification_level": 3
		},
		"group_access": {
			"access_level": 50,
			"notification_level": 3
		}
	}
}
`)

var project6Paylod = []byte(`
{
	"id": 6,
	"description": null,
	"default_branch": "master",
	"public": false,
	"visibility_level": 0,
	"ssh_url_to_repo": "git@example.com:brightbox/puppet.git",
	"http_url_to_repo": "http://example.com/brightbox/puppet.git",
	"web_url": "http://example.com/brightbox/puppet",
	"owner": {
	"id": 4,
		"name": "Brightbox",
		"created_at": "2013-09-30T13:46:02Z"
	},
	"name": "Puppet",
	"name_with_namespace": "Brightbox / Puppet",
	"path": "puppet",
	"path_with_namespace": "brightbox/puppet",
	"issues_enabled": true,
	"merge_requests_enabled": true,
	"wiki_enabled": true,
	"snippets_enabled": false,
	"created_at": "2013-09-30T13:46:02Z",
	"last_activity_at": "2013-09-30T13:46:02Z",
	"namespace": {
		"created_at": "2013-09-30T13:46:02Z",
		"description": "",
		"id": 4,
		"name": "Brightbox",
		"owner_id": 1,
		"path": "brightbox",
		"updated_at": "2013-09-30T13:46:02Z"
	},
	"archived": false,
	"permissions": {
		"project_access": {
			"access_level": 10,
			"notification_level": 3
		},
		"group_access": {
			"access_level": 50,
			"notification_level": 3
		}
	}
}
`)

// sample org list response
var sessionPayload = []byte(`
{
	"id": 1,
	"username": "john_smith",
	"email": "john@example.com",
	"name": "John Smith",
	"private_token": "dd34asd13as"
}
`)

// sample content response for .drone.yml request
var droneYamlPayload = []byte(`
{
	"type":     "file",
	"encoding": "base64",
	"name":     ".drone.yml",
	"path":     ".drone.yml",
	"content":  "aW1hZ2U6IGdv"
}
`)
