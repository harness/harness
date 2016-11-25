package fixtures

const HookPush = `
{
  "actor": {
    "username": "emmap1",
    "links": {
      "avatar": {
        "href": "https:\/\/bitbucket-api-assetroot.s3.amazonaws.com\/c\/photos\/2015\/Feb\/26\/3613917261-0-emmap1-avatar_avatar.png"
      }
    }
  },
  "repository": {
    "links": {
      "html": {
        "href": "https:\/\/api.bitbucket.org\/team_name\/repo_name"
      },
      "avatar": {
        "href": "https:\/\/api-staging-assetroot.s3.amazonaws.com\/c\/photos\/2014\/Aug\/01\/bitbucket-logo-2629490769-3_avatar.png"
      }
    },
    "full_name": "user_name\/repo_name",
    "scm": "git",
    "is_private": true
  },
  "push": {
    "changes": [
      {
        "new": {
          "type": "branch",
          "name": "name-of-branch",
          "target": {
            "type": "commit",
            "hash": "709d658dc5b6d6afcd46049c2f332ee3f515a67d",
            "author": {
              "raw": "emmap1 <email@domain.tld>",
              "username": "emmap1",
              "links": {
                "avatar": {
                  "href": "https:\/\/bitbucket-api-assetroot.s3.amazonaws.com\/c\/photos\/2015\/Feb\/26\/3613917261-0-emmap1-avatar_avatar.png"
                }
              }
            },
            "message": "new commit message\n",
            "date": "2015-06-09T03:34:49+00:00"
          }
        }
      }
    ]
  }
}
`

const HookPushEmptyHash = `
{
  "push": {
    "changes": [
      {
        "new": {
          "type": "branch",
          "target": { "hash": "" }
        }
      }
    ]
  }
}
`

const HookPull = `
{
  "actor": {
    "username": "emmap1",
    "links": {
      "avatar": {
        "href": "https:\/\/bitbucket-api-assetroot.s3.amazonaws.com\/c\/photos\/2015\/Feb\/26\/3613917261-0-emmap1-avatar_avatar.png"
      }
    }
  },
  "pullrequest": {
    "id": 1,
    "title": "Title of pull request",
    "description": "Description of pull request",
    "state": "OPEN",
    "author": {
      "username": "emmap1",
      "links": {
        "avatar": {
          "href": "https:\/\/bitbucket-api-assetroot.s3.amazonaws.com\/c\/photos\/2015\/Feb\/26\/3613917261-0-emmap1-avatar_avatar.png"
        }
      }
    },
    "source": {
      "branch": {
        "name": "branch2"
      },
      "commit": {
        "hash": "d3022fc0ca3d"
      },
      "repository": {
        "links": {
          "html": {
            "href": "https:\/\/api.bitbucket.org\/team_name\/repo_name"
          },
          "avatar": {
            "href": "https:\/\/api-staging-assetroot.s3.amazonaws.com\/c\/photos\/2014\/Aug\/01\/bitbucket-logo-2629490769-3_avatar.png"
          }
        },
        "full_name": "user_name\/repo_name",
        "scm": "git",
        "is_private": true
      }
    },
    "destination": {
      "branch": {
        "name": "master"
      },
      "commit": {
        "hash": "ce5965ddd289"
      },
      "repository": {
        "links": {
          "html": {
            "href": "https:\/\/api.bitbucket.org\/team_name\/repo_name"
          },
          "avatar": {
            "href": "https:\/\/api-staging-assetroot.s3.amazonaws.com\/c\/photos\/2014\/Aug\/01\/bitbucket-logo-2629490769-3_avatar.png"
          }
        },
        "full_name": "user_name\/repo_name",
        "scm": "git",
        "is_private": true
      }
    },
    "links": {
      "self": {
        "href": "https:\/\/api.bitbucket.org\/api\/2.0\/pullrequests\/pullrequest_id"
      },
      "html": {
        "href": "https:\/\/api.bitbucket.org\/pullrequest_id"
      }
    }
  },
  "repository": {
    "links": {
      "html": {
        "href": "https:\/\/api.bitbucket.org\/team_name\/repo_name"
      },
      "avatar": {
        "href": "https:\/\/api-staging-assetroot.s3.amazonaws.com\/c\/photos\/2014\/Aug\/01\/bitbucket-logo-2629490769-3_avatar.png"
      }
    },
    "full_name": "user_name\/repo_name",
    "scm": "git",
    "is_private": true
  }
}
`

const HookMerged = `
{
  "pullrequest": {
    "state": "MERGED"
  }
}
`
