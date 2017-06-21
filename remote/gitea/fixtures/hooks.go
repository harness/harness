package fixtures

// HookPush is a sample Gitea push hook
const HookPush = `
{
  "ref": "refs/heads/master",
  "before": "4b2626259b5a97b6b4eab5e6cca66adb986b672b",
  "after": "ef98532add3b2feb7a137426bba1248724367df5",
  "compare_url": "http://gitea.golang.org/gordon/hello-world/compare/4b2626259b5a97b6b4eab5e6cca66adb986b672b...ef98532add3b2feb7a137426bba1248724367df5",
  "commits": [
    {
      "id": "ef98532add3b2feb7a137426bba1248724367df5",
      "message": "bump\n",
      "url": "http://gitea.golang.org/gordon/hello-world/commit/ef98532add3b2feb7a137426bba1248724367df5",
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
    "html_url": "http://gitea.golang.org/gordon/hello-world",
    "ssh_url": "git@gitea.golang.org:gordon/hello-world.git",
    "clone_url": "http://gitea.golang.org/gordon/hello-world.git",
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
    "username": "gordon",
    "login": "gordon"
  },
  "sender": {
    "login": "gordon",
    "id": 1,
    "username": "gordon",
    "avatar_url": "http://gitea.golang.org///1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87"
  }
}
`

// HookPushTag is a sample Gitea tag hook
const HookPushTag = `{
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
    "html_url": "http://gitea.golang.org/gordon/hello-world",
    "ssh_url": "git@gitea.golang.org:gordon/hello-world.git",
    "clone_url": "http://gitea.golang.org/gordon/hello-world.git",
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

// HookPullRequest is a sample pull_request webhook payload
const HookPullRequest = `{
  "action": "opened",
  "number": 1,
  "pull_request": {
    "html_url": "http://gitea.golang.org/gordon/hello-world/pull/1",
    "state": "open",
    "title": "Update the README with new information",
    "body": "please merge",
    "user": {
      "id": 1,
      "username": "gordon",
      "full_name": "Gordon the Gopher",
      "email": "gordon@golang.org",
      "avatar_url": "http://gitea.golang.org///1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87"
    },
    "base_branch": "master",
    "base": {
      "label": "master",
      "ref": "master",
      "sha": "9353195a19e45482665306e466c832c46560532d"
    },
    "head_branch": "feature/changes",
    "head": {
      "label": "feature/changes",
      "ref": "feature/changes",
      "sha": "0d1a26e67d8f5eaf1f6ba5c57fc3c7d91ac0fd1c"
    }
  },
  "repository": {
    "id": 35129377,
    "name": "hello-world",
    "full_name": "gordon/hello-world",
    "owner": {
      "id": 1,
      "username": "gordon",
      "full_name": "Gordon the Gopher",
      "email": "gordon@golang.org",
      "avatar_url": "https://secure.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87"
    },
    "private": true,
    "html_url": "http://gitea.golang.org/gordon/hello-world",
    "clone_url": "https://gitea.golang.org/gordon/hello-world.git",
    "default_branch": "master"
  },
  "sender": {
      "id": 1,
      "login": "gordon",
      "username": "gordon",
      "full_name": "Gordon the Gopher",
      "email": "gordon@golang.org",
      "avatar_url": "https://secure.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87"
    }
}`
