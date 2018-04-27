// Copyright 2018 Drone.IO Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
    "email": "gordon@golang.org",
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

// HookPullRequest is a sample pull_request webhook payload
var HookPullRequest = `{
  "action": "opened",
  "number": 1,
  "pull_request": {
    "html_url": "http://gogs.golang.org/gordon/hello-world/pull/1",
    "state": "open",
    "title": "Update the README with new information",
    "body": "please merge",
    "user": {
      "id": 1,
      "username": "gordon",
      "full_name": "Gordon the Gopher",
      "email": "gordon@golang.org",
      "avatar_url": "http://gogs.golang.org///1.gravatar.com/avatar/8c58a0be77ee441bb8f8595b7f1b4e87"
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
    "html_url": "http://gogs.golang.org/gordon/hello-world",
    "clone_url": "https://gogs.golang.org/gordon/hello-world.git",
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
