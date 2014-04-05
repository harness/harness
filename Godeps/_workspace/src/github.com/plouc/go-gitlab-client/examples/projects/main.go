package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/plouc/go-gitlab-client"
	"io/ioutil"
	"os"
	"strconv"
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
	flag.StringVar(&method, "m", "", "Specify method to retrieve projects infos, available methods:\n"+
		"  > -m projects\n"+
		"  > -m project  -id PROJECT_ID\n"+
		"  > -m hooks    -id PROJECT_ID\n"+
		"  > -m branches -id PROJECT_ID\n"+
		"  > -m team     -id PROJECT_ID")

	var id string
	flag.StringVar(&id, "id", "", "Specify repository id")

	flag.Usage = func() {
		fmt.Printf("Usage:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *help == true || method == "" {
		flag.Usage()
		return
	}

	startedAt := time.Now()
	defer func() {
		fmt.Printf("processed in %v\n", time.Now().Sub(startedAt))
	}()

	switch method {
	case "projects":
		fmt.Println("Fetching projects…")

		projects, err := gitlab.Projects()
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		for _, project := range projects {
			fmt.Printf("> %6d | %s\n", project.Id, project.Name)
		}

	case "project":
		fmt.Println("Fetching project…")

		if id == "" {
			flag.Usage()
			return
		}

		project, err := gitlab.Project(id)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		format := "> %-23s: %s\n"

		fmt.Printf("%s\n", project.Name)
		fmt.Printf(format, "id", strconv.Itoa(project.Id))
		fmt.Printf(format, "name", project.Name)
		fmt.Printf(format, "description", project.Description)
		fmt.Printf(format, "default branch", project.DefaultBranch)
		fmt.Printf(format, "owner.name", project.Owner.Username)
		fmt.Printf(format, "public", strconv.FormatBool(project.Public))
		fmt.Printf(format, "path", project.Path)
		fmt.Printf(format, "path with namespace", project.PathWithNamespace)
		fmt.Printf(format, "issues enabled", strconv.FormatBool(project.IssuesEnabled))
		fmt.Printf(format, "merge requests enabled", strconv.FormatBool(project.MergeRequestsEnabled))
		fmt.Printf(format, "wall enabled", strconv.FormatBool(project.WallEnabled))
		fmt.Printf(format, "wiki enabled", strconv.FormatBool(project.WikiEnabled))
		fmt.Printf(format, "created at", project.CreatedAtRaw)
		//fmt.Printf(format, "namespace",           project.Namespace)

	case "branches":
		fmt.Println("Fetching project branches…")

		if id == "" {
			flag.Usage()
			return
		}

		branches, err := gitlab.ProjectBranches(id)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		for _, branch := range branches {
			fmt.Printf("> %s\n", branch.Name)
		}

	case "hooks":
		fmt.Println("Fetching project hooks…")

		if id == "" {
			flag.Usage()
			return
		}

		hooks, err := gitlab.ProjectHooks(id)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		for _, hook := range hooks {
			fmt.Printf("> [%d] %s, created on %s\n", hook.Id, hook.Url, hook.CreatedAtRaw)
		}

	case "team":
		fmt.Println("Fetching project team members…")

		if id == "" {
			flag.Usage()
			return
		}

		members, err := gitlab.ProjectMembers(id)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		for _, member := range members {
			fmt.Printf("> [%d] %s (%s) since %s\n", member.Id, member.Username, member.Name, member.CreatedAt)
		}
	}
}
