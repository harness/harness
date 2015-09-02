package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/bugagazavr/go-gitlab-client"
	"io/ioutil"
	"os"
	"time"
)

type Config struct {
	Host    string `json:"host"`
	ApiPath string `json:"api_path"`
	Token   string `json:"token"`
}

func main() {
	help := flag.Bool("help", false, "Show usage")

	file, e := ioutil.ReadFile("../config.json")
	if e != nil {
		fmt.Printf("Config file error: %v\n", e)
		os.Exit(1)
	}

	var config Config
	json.Unmarshal(file, &config)
	fmt.Printf("Results: %+v\n", config)

	gitlab := gogitlab.NewGitlab(config.Host, config.ApiPath, config.Token)

	var method string
	flag.StringVar(&method, "m", "", "Specify method to retrieve repositories, available methods:\n"+
		"  > branches\n"+
		"  > branch\n"+
		"  > tags\n"+
		"  > commits\n"+
		"  > commit_comments  -sha COMMIT_SHA\n"+
		"  > comment_a_commit -sha COMMIT_SHA -comment COMMENT_BODY")

	var id string
	flag.StringVar(&id, "id", "", "Specify repository id")

	var sha string
	flag.StringVar(&sha, "sha", "", "Specify commit sha")

	var comment string
	flag.StringVar(&comment, "comment", "", "The body of the new comment")

	flag.Usage = func() {
		fmt.Printf("Usage:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *help == true || method == "" || id == "" {
		flag.Usage()
		return
	}

	startedAt := time.Now()
	defer func() {
		fmt.Printf("processed in %v\n", time.Now().Sub(startedAt))
	}()

	switch method {
	case "branches":
		fmt.Println("Fetching repository branches…")

		branches, err := gitlab.RepoBranches(id)
		if err != nil {
			fmt.Println(err.Error())
		}

		for _, branch := range branches {
			fmt.Printf("> %s\n", branch.Name)
		}
	case "branch":
	case "tags":
		fmt.Println("Fetching repository tags…")

		tags, err := gitlab.RepoTags(id)
		if err != nil {
			fmt.Println(err.Error())
		}

		for _, tag := range tags {
			fmt.Printf("> %s\n", tag.Name)
		}
	case "commits":
		fmt.Println("Fetching repository commits…")

		commits, err := gitlab.RepoCommits(id)
		if err != nil {
			fmt.Println(err.Error())
		}

		for _, commit := range commits {
			fmt.Printf("(%s) %s > [%s] %s\n", commit.Id, commit.CreatedAt.Format("Mon 02 Jan 15:04"), commit.Author_Name, commit.Title)
		}
	case "commit_comments":
		fmt.Println("Fetching comments on a repository commit…")

		comments, err := gitlab.RepoCommitComments(id, sha)
		if err != nil {
			fmt.Println(err.Error())
		}

		for _, c := range comments {
			fmt.Printf("[%s] %s\n", c.Author.Username, c.Note)
		}
	case "comment_a_commit":
		fmt.Println("Sending a new comment on a repository commit…")

		c, err := gitlab.SendRepoCommitComment(id, sha, comment)
		if err != nil {
			fmt.Println(err.Error())
		}

		fmt.Printf("[%s] %s\n", c.Author.Username, c.Note)
	}
}
