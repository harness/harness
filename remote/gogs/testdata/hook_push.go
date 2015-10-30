package testdata

var PushHook = `
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
    "url": "http://gogs.golang.org/gordon/hello-world",
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
