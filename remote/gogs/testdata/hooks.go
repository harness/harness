package testdata

// X-Gogs-Event: pull_request
var PullRequestOpenHook = []byte(`
{
  "secret": "a_secret",
  "action": "opened",
  "number": 1,
  "pull_request": {
    "id": 2,
    "number": 1,
    "user": {
      "id": 1,
      "username": "strk",
      "full_name": "",
      "email": "strk@kbt.io",
      "avatar_url": "https://avatars.kbt.io/avatar/fe2a9e759730ee64c44bf8901bf4ccc3"
    },
    "title": "dot",
    "body": "could you figure",
    "labels": [],
    "milestone": null,
    "assignee": null,
    "state": "open",
    "comments": 0,
    "html_url": "http://cdb:3000/org1/test3/pulls/1",
    "mergeable": true,
    "merged": false,
    "merged_at": null,
    "merge_commit_sha": null,
    "merged_by": null
  },
  "repository": {
    "id": 5,
    "owner": {
      "id": 5,
      "username": "org1",
      "full_name": "org1",
      "email": "",
      "avatar_url": "http://cdb:3000/avatars/5"
    },
    "name": "test3",
    "full_name": "org1/test3",
    "description": "just a test",
    "private": false,
    "fork": false,
    "html_url": "http://cdb:3000/org1/test3",
    "ssh_url": "strk@git.osgeo.org:org1/test3.git",
    "clone_url": "http://cdb:3000/org1/test3.git",
    "website": "",
    "stars_count": 0,
    "forks_count": 1,
    "watchers_count": 2,
    "open_issues_count": 0,
    "default_branch": "master",
    "created_at": "2016-08-31T22:45:16+02:00",
    "updated_at": "2016-08-31T22:45:31+02:00"
  },
  "sender": {
    "id": 1,
    "username": "strk",
    "full_name": "",
    "email": "strk@kbt.io",
    "avatar_url": "https://avatars.kbt.io/avatar/fe2a9e759730ee64c44bf8901bf4ccc3"
  }
}
')

// X-Gogs-Event: pull_request
var PullRequestSynchronizeHook = []byte(`
{
  "secret": "a_secret",
  "action": "synchronized",
  "number": 1,
  "pull_request": {
    "id": 2,
    "number": 1,
    "user": {
      "id": 1,
      "username": "strk",
      "full_name": "",
      "email": "strk@kbt.io",
      "avatar_url": "https://avatars.kbt.io/avatar/fe2a9e759730ee64c44bf8901bf4ccc3"
    },
    "title": "dot",
    "body": "could you figure",
    "labels": [],
    "milestone": null,
    "assignee": null,
    "state": "open",
    "comments": 0,
    "html_url": "http://cdb:3000/org1/test3/pulls/1",
    "mergeable": true,
    "merged": false,
    "merged_at": null,
    "merge_commit_sha": null,
    "merged_by": null
  },
  "repository": {
    "id": 5,
    "owner": {
      "id": 5,
      "username": "org1",
      "full_name": "org1",
      "email": "",
      "avatar_url": "http://cdb:3000/avatars/5"
    },
    "name": "test3",
    "full_name": "org1/test3",
    "description": "just a test",
    "private": false,
    "fork": false,
    "html_url": "http://cdb:3000/org1/test3",
    "ssh_url": "strk@git.osgeo.org:org1/test3.git",
    "clone_url": "http://cdb:3000/org1/test3.git",
    "website": "",
    "stars_count": 0,
    "forks_count": 1,
    "watchers_count": 2,
    "open_issues_count": 0,
    "default_branch": "master",
    "created_at": "2016-08-31T22:45:16+02:00",
    "updated_at": "2016-08-31T22:45:31+02:00"
  },
  "sender": {
    "id": 1,
    "username": "strk",
    "full_name": "",
    "email": "strk@kbt.io",
    "avatar_url": "https://avatars.kbt.io/avatar/fe2a9e759730ee64c44bf8901bf4ccc3"
  }
}
')

// X-Gogs-Event: push
var PushHook = []byte(`
{
  "secret": "a_secret",
  "ref": "refs/heads/master",
  "before": "117b1990205dbc6395656ef1ed2125719aa7f4d3",
  "after": "7d7605add378b55e6154d96b3e0957d392e2cc14",
  "compare_url": "http://cdb:3000/org1/test3/compare/117b1990205dbc6395656ef1ed2125719aa7f4d3...7d7605add378b55e6154d96b3e0957d392e2cc14",
  "commits": [
    {
      "id": "7d7605add378b55e6154d96b3e0957d392e2cc14",
      "message": "Capitalize\n",
      "url": "http://cdb:3000/org1/test3/commit/7d7605add378b55e6154d96b3e0957d392e2cc14",
      "author": {
        "name": "Sandro Santilli",
        "email": "strk@kbt.io",
        "username": "strk"
      },
      "committer": {
        "name": "Sandro Santilli",
        "email": "strk@kbt.io",
        "username": "strk"
      },
      "timestamp": "2016-08-31T22:51:59+02:00"
    },
    {
      "id": "85800d8ecf8107626dc43a0cbdf218c31cd04779",
      "message": "dot\n",
      "url": "http://cdb:3000/org1/test3/commit/85800d8ecf8107626dc43a0cbdf218c31cd04779",
      "author": {
        "name": "Sandro Santilli",
        "email": "strk@kbt.io",
        "username": "strk"
      },
      "committer": {
        "name": "Sandro Santilli",
        "email": "strk@kbt.io",
        "username": "strk"
      },
      "timestamp": "2016-08-31T22:46:53+02:00"
    }
  ],
  "repository": {
    "id": 5,
    "owner": {
      "id": 5,
      "username": "org1",
      "full_name": "org1",
      "email": "",
      "avatar_url": "http://cdb:3000/avatars/5"
    },
    "name": "test3",
    "full_name": "org1/test3",
    "description": "just a test",
    "private": false,
    "fork": false,
    "html_url": "http://cdb:3000/org1/test3",
    "ssh_url": "strk@git.osgeo.org:org1/test3.git",
    "clone_url": "http://cdb:3000/org1/test3.git",
    "website": "",
    "stars_count": 0,
    "forks_count": 1,
    "watchers_count": 2,
    "open_issues_count": 0,
    "default_branch": "master",
    "created_at": "2016-08-31T22:45:16+02:00",
    "updated_at": "2016-08-31T22:45:31+02:00"
  },
  "pusher": {
    "id": 1,
    "username": "strk",
    "full_name": "",
    "email": "strk@kbt.io",
    "avatar_url": "https://avatars.kbt.io/avatar/fe2a9e759730ee64c44bf8901bf4ccc3"
  },
  "sender": {
    "id": 1,
    "username": "strk",
    "full_name": "",
    "email": "strk@kbt.io",
    "avatar_url": "https://avatars.kbt.io/avatar/fe2a9e759730ee64c44bf8901bf4ccc3"
  }
}
')
