package fixtures

// Sample Gogs push hook
var HookPush = `
{
  "ref": "refs/heads/master",
  "before": "4b2626259b5a97b6b4eab5e6cca66adb986b672b",
  "after": "ef98532add3b2feb7a137426bba1248724367df5",
  "compare_url": "http://gogs.golang.org/gordon/hello-world/compare/4b2626259b5a97b6b4eab5e6cca66adb986b672b...ef98532add3b2feb7a137426bba1248724367df5",
  "commits": [
    {
      "id": "ef98532add3b2feb7a137426bba1248724367df5",
      "message": "bump\n",
      "url": "http://gogs.golang.org/gordon/hello-world/commit/ef98532add3b2feb7a137426bba1248724367df5",
      "author": {
        "name": "Gordon the Gopher",
        "email": "gordon@golang.org",
        "username": "gordon"
      }
    }
  ],
  "repository": {
    "id": 1,
    "name": "hello-world",
    "full_name": "gordon/hello-world",
    "html_url": "http://gogs.golang.org/gordon/hello-world",
    "ssh_url": "git@gogs.golang.org:gordon/hello-world.git",
    "clone_url": "http://gogs.golang.org/gordon/hello-world.git",
    "description": "",
    "website": "",
    "watchers": 1,
    "owner": {
      "name": "gordon",
      "email": "gordon@golang.org",
      "username": "gordon"
    },
    "private": true
  },
  "pusher": {
    "name": "gordon",
    "email": "gordon@golang.org",
    "username": "gordon"
  },
  "sender": {
    "login": "gordon",
    "id": 1,
    "avatar_url": "http://gogs.golang.org///1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87"
  }
}
`

// Sample Gogs tag hook
var HookPushTag = `{
  "secret": "l26Un7G7HXogLAvsyf2hOA4EMARSTsR3",
  "ref": "v1.0.0",
  "ref_type": "tag",
  "repository": {
    "id": 1,
    "owner": {
      "id": 1,
      "username": "gordon",
      "full_name": "Gordon the Gopher",
      "email": "gordon@golang.org",
      "avatar_url": "https://secure.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87"
    },
    "name": "hello-world",
    "full_name": "gordon/hello-world",
    "description": "",
    "private": true,
    "fork": false,
    "html_url": "http://gogs.golang.org/gordon/hello-world",
    "ssh_url": "git@gogs.golang.org:gordon/hello-world.git",
    "clone_url": "http://gogs.golang.org/gordon/hello-world.git",
    "default_branch": "master",
    "created_at": "2015-10-22T19:32:44Z",
    "updated_at": "2016-11-24T13:37:16Z"
  },
  "sender": {
    "id": 1,
    "username": "gordon",
    "full_name": "Gordon the Gopher",
    "email": "gordon@golang.org",
    "avatar_url": "https://secure.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87"
  }
}`
