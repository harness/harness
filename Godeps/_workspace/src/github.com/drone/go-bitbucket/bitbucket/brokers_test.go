package bitbucket

import (
	"testing"
)

/*
func Test_Brokers(t *testing.T) {

	// CREATE a broker
	s, err := client.Brokers.Create(testUser, testRepo, "https://bitbucket.org/post")
	if err != nil {
		t.Error(err)
	}

	// DELETE (deferred)
	defer client.Brokers.Delete(testUser, testRepo, s.Id)

	// FIND the broker by id
	broker, err := client.Brokers.Find(testUser, testRepo, s.Id)
	if err != nil {
		t.Error(err)
	}

	// verify we get back the correct data
	if broker.Id != s.Id {
		t.Errorf("broker id [%v]; want [%v]", broker.Id, s.Id)
	}
	if broker.Profile.Type != s.Profile.Type {
		t.Errorf("broker type [%v]; want [%v]", broker.Profile.Type, s.Profile.Type)
	}
	if broker.Profile.Fields[0].Value != "https://bitbucket.org/post" {
		t.Errorf("broker url [%v]; want [%v]", broker.Profile.Fields[0].Value, "https://bitbucket.org/post")
	}

	// UPDATE the broker
	_, err = client.Brokers.Update(testUser, testRepo, "https://bitbucket.org/post2", s.Id)
	if err != nil {
		t.Error(err)
	}

	// LIST the brokers
	brokers, err := client.Brokers.List(testUser, testRepo)
	if err != nil {
		t.Error(err)
	}

	if len(brokers) == 0 {
		t.Errorf("List of brokers returned empty set")
	}
}
*/
func Test_PostReceiveHooks(t *testing.T) {
	hook, err := ParseHook([]byte(sampleHook))
	if err != nil {
		t.Error(err)
		return
	}

	if hook.Commits[0].Branch != "master" {
		t.Errorf("expected branch [%v]; got [%v]", "master", hook.Commits[0].Branch)
	}

	if hook.Repo.Owner != "marcus" {
		t.Errorf("expected branch [%v]; got [%v]", "marcus", hook.Repo.Owner)
	}

	if hook.Repo.Slug != "project-x" {
		t.Errorf("expected slug [%v]; got [%v]", "project-x", hook.Repo.Slug)
	}

	// What happens if we get a hook from, say, Google Code?
	hook, err = ParseHook([]byte(invalidHook))
	if err == nil {
		t.Errorf("Expected error parsing Google Code hook")
		return
	}
}

func Test_IsValidSender(t *testing.T) {
	str := map[string]bool{
		"63.246.22.222": true,
		"127.0.0.1":     false,
		"localhost":     false,
		"1.2.3.4":       false,
	}

	for k, v := range str {
		if IsValidSender(k) != v {
			t.Errorf("expected IP address [%v] validation [%v]", k, v)
		}
	}
}

var sampleHook = `
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

var invalidHook = `
{
  "repository_path": "https:\/\/code.google.com\/p\/rydzewski-hg\/",
  "project_name": "rydzewski-hg",
  "revisions": [
    {
      "added": [
        "\/README"
      ],
      "parents": [

      ],
      "author": "John Doe <john@example.com>",
      "url": "http:\/\/rydzewski-hg.googlecode.com\/hg-history\/be12639f52f33b0861e647d3a795f863061395bf\/",
      "timestamp": 1345764974,
      "message": "testing ...",
      "path_count": 1,
      "removed": [

      ],
      "modified": [

      ],
      "revision": "be12639f52f33b0861e647d3a795f863061395bf"
    }
  ],
  "revision_count": 1
}
`
