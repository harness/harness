package testdata

// sample repository list
var allProjectsPayload = []byte(`
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
			"username": "some_user",
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
			"id": 1,
			"name": "Brightbox",
			"username": "test_user",
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
		"archived": true
	}
]
`)

var notArchivedProjectsPayload = []byte(`
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
			"username": "some_user",
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
		"username": "some_user",
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
		"id": 1,
		"name": "Brightbox",
		"username": "test_user",
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
		"project_access": null,
		"group_access": null
	}
}
`)
