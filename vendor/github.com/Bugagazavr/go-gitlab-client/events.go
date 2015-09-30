package gogitlab

import (
	"encoding/xml"
	"fmt"
	"time"
)

type Person struct {
	Name  string `xml:"name"json:"name"`
	Email string `xml:"email"json:"email"`
}

type Link struct {
	Rel  string `xml:"rel,attr,omitempty"json:"rel"`
	Href string `xml:"href,attr"json:"href"`
}

type ActivityFeed struct {
	Title   string        `xml:"title"json:"title"`
	Id      string        `xml:"id"json:"id"`
	Link    []Link        `xml:"link"json:"link"`
	Updated time.Time     `xml:"updated,attr"json:"updated"`
	Entries []*FeedCommit `xml:"entry"json:"entries"`
}

type FeedCommit struct {
	Id      string    `xml:"id"json:"id"`
	Title   string    `xml:"title"json:"title"`
	Link    []Link    `xml:"link"json:"link"`
	Updated time.Time `xml:"updated"json:"updated"`
	Author  Person    `xml:"author"json:"author"`
	Summary string    `xml:"summary"json:"summary"`
	//<media:thumbnail width="40" height="40" url="https://secure.gravatar.com/avatar/7070eab7c6206530d3b7820362227fec?s=40&amp;d=mm"/>
}

func (g *Gitlab) Activity() (ActivityFeed, error) {

	url := g.BaseUrl + dasboard_feed_path + "?private_token=" + g.Token
	fmt.Println(url)

	contents, err := g.buildAndExecRequest("GET", url, nil)
	if err != nil {
		fmt.Println("%s", err)
	}

	var activity ActivityFeed
	err = xml.Unmarshal(contents, &activity)
	if err != nil {
		fmt.Println("%s", err)
	}

	return activity, err
}

func (g *Gitlab) RepoActivityFeed(feedPath string) ActivityFeed {

	url := g.BaseUrl + g.RepoFeedPath + "?private_token=" + g.Token

	contents, err := g.buildAndExecRequest("GET", url, nil)
	if err != nil {
		fmt.Println("%s", err)
	}

	var activity ActivityFeed
	err = xml.Unmarshal(contents, &activity)
	if err != nil {
		fmt.Println("%s", err)
	}

	return activity
}
