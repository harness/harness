package github

import (
	"strings"
	"testing"
)

func Test_Hooks(t *testing.T) {

	// CREATE a hook
	h, err := client.Hooks.Create(testUser, testRepo, "https://localhost.com/post")
	if err != nil {
		t.Error(err)
		return
	}

	// DELETE (deferred)
	defer client.Hooks.Delete(testUser, testRepo, h.Id)

	// FIND the hook by id
	hook, err := client.Hooks.Find(testUser, testRepo, h.Id)
	if err != nil {
		t.Error(err)
		return
	}

	if hook.Config.Url != "https://localhost.com/post" {
		t.Errorf("hook url [%v]; want [%v]", hook.Config.Url, h.Config.Url)
	}

	// UPDATE the hook
	hook.Config.Url = "https://localhost.com/post2"
	hook2, err := client.Hooks.Update(testUser, testRepo, hook)
	if err != nil {
		t.Error(err)
		return
	}

	if hook.Config.Url != hook2.Config.Url {
		t.Errorf("updated hook url [%v]; want [%v]", hook.Config.Url, hook2.Config.Url)
	}

	// LIST the hooks
	hooks, err := client.Hooks.List(testUser, testRepo)
	if err != nil {
		t.Error(err)
		return
	}

	if len(hooks) == 0 {
		t.Errorf("List of hooks returned empty set")
	}

}


func Test_PostReceiveHooks(t *testing.T) {
	hook, err := ParseHook([]byte(sampleHook))
	if err != nil {
		t.Error(err)
		return
	}

	if hook.IsGithubPages() {
		t.Errorf("expected build hook ref not github pages")
	}
	if hook.IsTag() {
		t.Errorf("expected build hook ref not a tag")
	}
	if hook.IsHead() == false {
		t.Errorf("expected build hook ref labeled as head")
	}
	if hook.Branch() != "master" {
		t.Errorf("expected branch [%v]; got [%v]", "master", hook.Branch())
	}
	if hook.IsDeleted() {
		t.Errorf("expected deleted == false")
	}

	// Test Tag detection

	tagHook := strings.Replace(sampleHook, "refs/heads/master", "refs/tags/mytag", -1)
	hook, err = ParseHook([]byte(tagHook))
	if err != nil {
		t.Error(err)
		return
	}

	if hook.IsTag() == false {
		t.Errorf("expected build hook ref IS a tag")
	}

	if hook.IsHead() == true {
		t.Errorf("expected build hook ref NOT labeled as head")
	}
	if hook.IsDeleted() {
		t.Errorf("expected deleted == false")
	}

	// Test Github Pages detection

	ghpagesHook := strings.Replace(sampleHook, "refs/heads/master", "refs/heads/gh-pages", -1)
	hook, err = ParseHook([]byte(ghpagesHook))
	if err != nil {
		t.Error(err)
		return
	}

	if hook.IsGithubPages() == false {
		t.Errorf("expected build hook ref IS github pages branch")
	}

	// Now let's make sure things don't choke if we try
	// to parse a Bitbucket URL
	hook, err = ParseHook([]byte(sampleBitbucketHook))
	if err == nil && err != ErrInvalidPostReceiveHook {
		t.Errorf("expected error parsing Bitbucket Hook")
	}

	// Test Branch Deleted

	deletedHook := strings.Replace(sampleHook, `"deleted":false`, `"deleted":true`, -1)
	hook, err = ParseHook([]byte(deletedHook))
	if err != nil {
		t.Error(err)
		return
	}

	if hook.IsDeleted() == false {
		t.Errorf("expected deleted == true")
	}


}


func Test_IsValidSender(t *testing.T) {
	str := map[string]bool {
		"207.97.227.253"  : true,
		"50.57.128.197"   : true,
		"108.171.174.178" : true,
		"50.57.231.61"    : true,
		"127.0.0.1" : false,
		"localhost" : false,
		"1.2.3.4"   : false,
	}

	for k, v := range str {
		if IsValidSender(k) != v {
			t.Errorf("expected IP address [%v] validation [%v]", k, v)
		}
	}
}

var sampleHook = `
{
  "before": "5aef35982fb2d34e9d9d4502f6ede1072793222d",
  "repository": {
    "url": "http://github.com/defunkt/github",
    "name": "github",
    "description": "You're lookin' at it.",
    "watchers": 5,
    "forks": 2,
    "private": 1,
    "owner": {
      "email": "chris@ozmm.org",
      "name": "defunkt"
    }
  },
  "commits": [
    {
      "id": "41a212ee83ca127e3c8cf465891ab7216a705f59",
      "url": "http://github.com/defunkt/github/commit/41a212ee83ca127e3c8cf465891ab7216a705f59",
      "author": {
        "email": "chris@ozmm.org",
        "name": "Chris Wanstrath"
      },
      "message": "okay i give in",
      "timestamp": "2008-02-15T14:57:17-08:00",
      "added": ["filepath.rb"]
    },
    {
      "id": "de8251ff97ee194a289832576287d6f8ad74e3d0",
      "url": "http://github.com/defunkt/github/commit/de8251ff97ee194a289832576287d6f8ad74e3d0",
      "author": {
        "email": "chris@ozmm.org",
        "name": "Chris Wanstrath"
      },
      "message": "update pricing a tad",
      "timestamp": "2008-02-15T14:36:34-08:00"
    }
  ],
  "after": "de8251ff97ee194a289832576287d6f8ad74e3d0",
  "ref": "refs/heads/master",
  "deleted":false
}
`

var sampleBitbucketHook = `

{
    "canon_url": "https://bitbucket.org", 
    "commits": [
        {
            "author": "marcus", 
            "branch": "master", 
            "files": [
                {
                    "file": "somefile.py", 
                    "type": "modified"
                }
            ], 
            "message": "Added some more things to somefile.py\n", 
            "node": "620ade18607a", 
            "parents": [
                "702c70160afc"
            ], 
            "raw_author": "Marcus Bertrand <marcus@somedomain.com>", 
            "raw_node": "620ade18607ac42d872b568bb92acaa9a28620e9", 
            "revision": null, 
            "size": -1, 
            "timestamp": "2012-05-30 05:58:56", 
            "utctimestamp": "2012-05-30 03:58:56+00:00"
        }
    ], 
    "repository": {
        "absolute_url": "/marcus/project-x/", 
        "fork": false, 
        "is_private": true, 
        "name": "Project X", 
        "owner": "marcus", 
        "scm": "git", 
        "slug": "project-x", 
        "website": "https://atlassian.com/"
    }, 
    "user": "marcus"
}
`
