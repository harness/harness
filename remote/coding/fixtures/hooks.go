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

const PushHook = `
{
    "ref": "refs/heads/master",
    "before": "861f2315056e8925e627a6f46518b9df05896e24",
    "commits": [
        {
            "committer": {
                "name": "demo1",
                "email": "demo1@gmail.com"
            },
            "web_url": "https://coding.net/u/demo1/p/test1/git/commit/5b9912a6ff272e9c93a4c44c278fe9b359ed1ab4",
            "short_message": "new file .drone.yml\n",
            "sha": "5b9912a6ff272e9c93a4c44c278fe9b359ed1ab4"
        }
    ],
    "after": "5b9912a6ff272e9c93a4c44c278fe9b359ed1ab4",
    "event": "push",
    "repository": {
        "owner": {
            "path": "/u/demo1",
            "web_url": "https://coding.net/u/demo1",
            "global_key": "demo1",
            "name": "demo1",
            "avatar": "/static/fruit_avatar/Fruit-20.png"
        },
        "https_url": "https://git.coding.net/demo1/test1.git",
        "web_url": "https://coding.net/u/demo1/p/test1",
        "project_id": "99999999",
        "ssh_url": "git@git.coding.net:demo1/test1.git",
        "name": "test1",
        "description": ""
    },
    "user": {
        "path": "/u/demo1",
        "web_url": "https://coding.net/u/demo1",
        "global_key": "demo1",
        "name": "demo1",
        "avatar": "/static/fruit_avatar/Fruit-20.png"
    }
}
`

const DeleteBranchPushHook = `
{
    "ref": "refs/heads/master",
    "before": "861f2315056e8925e627a6f46518b9df05896e24",
    "after": "0000000000000000000000000000000000000000",
    "event": "push",
    "repository": {
        "owner": {
            "path": "/u/demo1",
            "web_url": "https://coding.net/u/demo1",
            "global_key": "demo1",
            "name": "demo1",
            "avatar": "/static/fruit_avatar/Fruit-20.png"
        },
        "https_url": "https://git.coding.net/demo1/test1.git",
        "web_url": "https://coding.net/u/demo1/p/test1",
        "project_id": "99999999",
        "ssh_url": "git@git.coding.net:demo1/test1.git",
        "name": "test1",
        "description": ""
    },
    "user": {
        "path": "/u/demo1",
        "web_url": "https://coding.net/u/demo1",
        "global_key": "demo1",
        "name": "demo1",
        "avatar": "/static/fruit_avatar/Fruit-20.png"
    }
}
`

const PullRequestHook = `
{
    "pull_request": {
        "target_branch": "master",
        "title": "pr1",
        "body": "pr message",
        "source_sha": "",
        "source_repository": {
            "owner": {
                "path": "/u/demo2",
                "web_url": "https://coding.net/u/demo2",
                "global_key": "demo2",
                "name": "demo2",
                "avatar": "/static/fruit_avatar/Fruit-2.png"
            },
            "https_url": "https://git.coding.net/demo2/test2.git",
            "web_url": "https://coding.net/u/demo2/p/test2",
            "project_id": "7777777",
            "ssh_url": "git@git.coding.net:demo2/test2.git",
            "name": "test2",
            "description": "",
            "git_url": "git://git.coding.net/demo2/test2.git"
        },
        "source_branch": "master",
        "number": 1,
        "web_url": "https://coding.net/u/demo1/p/test2/git/pull/1",
        "merge_commit_sha": "55e77b328b71d3ee4f9e70a5f67231b0acceeadc",
        "target_sha": "",
        "action": "create",
        "id": 7586,
        "user": {
            "path": "/u/demo2",
            "web_url": "https://coding.net/u/demo2",
            "global_key": "demo2",
            "name": "demo2",
            "avatar": "/static/fruit_avatar/Fruit-2.png"
        },
        "status": "CANMERGE"
    },
    "repository": {
        "owner": {
            "path": "/u/demo1",
            "web_url": "https://coding.net/u/demo1",
            "global_key": "demo1",
            "name": "demo1",
            "avatar": "/static/fruit_avatar/Fruit-20.png"
        },
        "https_url": "https://git.coding.net/demo1/test2.git",
        "web_url": "https://coding.net/u/demo1/p/test2",
        "project_id": "6666666",
        "ssh_url": "git@git.coding.net:demo1/test2.git",
        "name": "test2",
        "description": "",
        "git_url": "git://git.coding.net/demo1/test2.git"
    },
    "event": "pull_request"
}
`

const MergeRequestHook = `
{
    "merge_request": {
        "target_branch": "master",
        "title": "mr1",
        "body": "<p>mr message</p>",
        "source_sha": "",
        "source_branch": "branch1",
        "number": 1,
        "web_url": "https://coding.net/u/demo1/p/test1/git/merge/1",
        "merge_commit_sha": "74e6755580c34e9fd81dbcfcbd43ee5f30259436",
        "target_sha": "",
        "action": "create",
        "id": 533428,
        "user": {
            "path": "/u/demo1",
            "web_url": "https://coding.net/u/demo1",
            "global_key": "demo1",
            "name": "demo1",
            "avatar": "/static/fruit_avatar/Fruit-20.png"
        },
        "status": "CANMERGE"
    },
    "repository": {
        "owner": {
            "path": "/u/demo1",
            "web_url": "https://coding.net/u/demo1",
            "global_key": "demo1",
            "name": "demo1",
            "avatar": "/static/fruit_avatar/Fruit-20.png"
        },
        "https_url": "https://git.coding.net/demo1/test1.git",
        "web_url": "https://coding.net/u/demo1/p/test1",
        "project_id": "99999999",
        "ssh_url": "git@git.coding.net:demo1/test1.git",
        "name": "test1",
        "description": ""
    },
    "event": "merge_request"
}
`
