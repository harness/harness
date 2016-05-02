package server

import (
	"strings"

	"github.com/drone/drone/store"

	"github.com/gin-gonic/gin"
)

const (
	slashDeploy  = "deploy"
	slashRestart = "restart"
	slashStatus  = "status"
)

// Slack is handler function that handles Slack slash commands.
func Slack(c *gin.Context) {
	command := c.Param("command")
	text := c.PostForm("text")
	args := strings.Split(text, " ")

	if command == "" {
		command = args[0]
		args = args[1:]
	}

	switch command {
	case slashStatus:
		slackStatus(c, args)

	case slashRestart:
		slackRestart(c, args)

	case slashDeploy:
		slackDeploy(c, args)

	default:
		c.String(200, "sorry, I didn't understand [%s]", text)
	}
}

func slackDeploy(c *gin.Context, args []string) {
	if len(args) != 3 {
		c.String(200, "Invalid command. Please provide [repo] [build number] [environment]")
		return
	}
	var (
		repo = args[0]
		num  = args[1]
		env  = args[2]
	)
	owner, name, _ := parseRepoBranch(repo)

	c.String(200, "deploying build %s/%s#%s to %s", owner, name, num, env)
}

func slackRestart(c *gin.Context, args []string) {
	var (
		repo = args[0]
		num  = args[1]
	)
	owner, name, _ := parseRepoBranch(repo)

	c.String(200, "restarting build %s/%s#%s", owner, name, num)
}

func slackStatus(c *gin.Context, args []string) {
	var (
		owner  string
		name   string
		branch string
	)
	if len(args) > 0 {
		owner, name, branch = parseRepoBranch(args[0])
	}

	repo, err := store.GetRepoOwnerName(c, owner, name)
	if err != nil {
		c.String(200, "cannot find repository %s/%s", owner, name)
		return
	}
	if branch == "" {
		branch = repo.Branch
	}
	build, err := store.GetBuildLast(c, repo, branch)
	if err != nil {
		c.String(200, "cannot find status for %s/%s@%s", owner, name, branch)
		return
	}
	c.String(200, "%s@%s build number %d finished with status %s",
		repo.FullName,
		build.Branch,
		build.Number,
		build.Status,
	)
}

func parseRepoBranch(repo string) (owner, name, branch string) {

	parts := strings.Split(repo, "@")
	if len(parts) == 2 {
		branch = parts[1]
		repo = parts[0]
	}

	parts = strings.Split(repo, "/")
	if len(parts) == 2 {
		owner = parts[0]
		name = parts[1]
	}
	return owner, name, branch
}
