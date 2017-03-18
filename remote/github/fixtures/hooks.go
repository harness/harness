package fixtures

// HookPush is a sample push hook.
// https://developer.github.com/v3/activity/events/types/#pushevent
const HookPush = `
{
  "ref": "refs/heads/changes",
  "created": false,
  "deleted": false,
  "head_commit": {
    "id": "0d1a26e67d8f5eaf1f6ba5c57fc3c7d91ac0fd1c",
    "message": "Update README.md",
    "timestamp": "2015-05-05T19:40:15-04:00",
    "url": "https://github.com/baxterthehacker/public-repo/commit/0d1a26e67d8f5eaf1f6ba5c57fc3c7d91ac0fd1c",
    "author": {
      "name": "baxterthehacker",
      "email": "baxterthehacker@users.noreply.github.com",
      "username": "baxterthehacker"
    },
    "committer": {
      "name": "baxterthehacker",
      "email": "baxterthehacker@users.noreply.github.com",
      "username": "baxterthehacker"
    }
  },
  "repository": {
    "id": 35129377,
    "name": "public-repo",
    "full_name": "baxterthehacker/public-repo",
    "owner": {
      "name": "baxterthehacker",
      "email": "baxterthehacker@users.noreply.github.com"
    },
    "private": false,
    "html_url": "https://github.com/baxterthehacker/public-repo",
    "default_branch": "master"
  },
  "pusher": {
    "name": "baxterthehacker",
    "email": "baxterthehacker@users.noreply.github.com"
  },
  "sender": {
    "login": "baxterthehacker",
    "avatar_url": "https://avatars.githubusercontent.com/u/6752317?v=3"
  }
}
`

// HookPush is a sample push hook that is marked as deleted, and is expected to
// be ignored.
const HookPushDeleted = `
{
  "deleted": true
}
`

// HookPullRequest is a sample hook pull request
// https://developer.github.com/v3/activity/events/types/#pullrequestevent
const HookPullRequest = `
{
  "action": "opened",
  "number": 1,
  "pull_request": {
    "url": "https://api.github.com/repos/baxterthehacker/public-repo/pulls/1",
    "html_url": "https://github.com/baxterthehacker/public-repo/pull/1",
    "number": 1,
    "state": "open",
    "title": "Update the README with new information",
    "user": {
      "login": "baxterthehacker",
      "avatar_url": "https://avatars.githubusercontent.com/u/6752317?v=3"
    },
    "base": {
      "label": "baxterthehacker:master",
      "ref": "master",
      "sha": "9353195a19e45482665306e466c832c46560532d"
    },
    "head": {
      "label": "baxterthehacker:changes",
      "ref": "changes",
      "sha": "0d1a26e67d8f5eaf1f6ba5c57fc3c7d91ac0fd1c"
    }
  },
  "repository": {
    "id": 35129377,
    "name": "public-repo",
    "full_name": "baxterthehacker/public-repo",
    "owner": {
      "login": "baxterthehacker",
      "avatar_url": "https://avatars.githubusercontent.com/u/6752317?v=3"
    },
    "private": true,
    "html_url": "https://github.com/baxterthehacker/public-repo",
    "clone_url": "https://github.com/baxterthehacker/public-repo.git",
    "default_branch": "master"
  },
  "sender": {
    "login": "octocat",
    "avatar_url": "https://avatars.githubusercontent.com/u/6752317?v=3"
  }
}
`

// HookPullRequestInvalidAction is a sample hook pull request that has an
// action not equal to synchrize or opened, and is expected to be ignored.
const HookPullRequestInvalidAction = `
{
  "action": "reopened",
  "number": 1
}
`

// HookPullRequestInvalidState is a sample hook pull request that has a state
// not equal to open, and is expected to be ignored.
const HookPullRequestInvalidState = `
{
  "action": "synchronize",
  "pull_request": {
    "number": 1,
    "state": "closed"
  }
}
`

// HookPush is a sample deployment hook.
// https://developer.github.com/v3/activity/events/types/#deploymentevent
const HookDeploy = `
{
  "deployment": {
    "url": "https://api.github.com/repos/baxterthehacker/public-repo/deployments/710692",
    "id": 710692,
    "sha": "9049f1265b7d61be4a8904a9a27120d2064dab3b",
    "ref": "master",
    "task": "deploy",
    "payload": {
    },
    "environment": "production",
    "description": null,
    "creator": {
      "login": "baxterthehacker",
      "avatar_url": "https://avatars.githubusercontent.com/u/6752317?v=3"
    }
  },
  "repository": {
    "id": 35129377,
    "name": "public-repo",
    "full_name": "baxterthehacker/public-repo",
    "owner": {
      "login": "baxterthehacker",
      "avatar_url": "https://avatars.githubusercontent.com/u/6752317?v=3"
    },
    "private": true,
    "html_url": "https://github.com/baxterthehacker/public-repo",
    "clone_url": "https://github.com/baxterthehacker/public-repo.git",
    "default_branch": "master"
  },
  "sender": {
    "login": "baxterthehacker",
    "avatar_url": "https://avatars.githubusercontent.com/u/6752317?v=3"
  }
}
`
