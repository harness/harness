package handler

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/drone/drone/pkg/database"
	. "github.com/drone/drone/pkg/model"
	"github.com/drone/drone/pkg/queue"
	"github.com/plouc/go-gitlab-client"
)

type GitlabHandler struct {
	queue   *queue.Queue
	apiPath string
}

func NewGitlabHandler(queue *queue.Queue) *GitlabHandler {
	return &GitlabHandler{
		queue:   queue,
		apiPath: "/api/v3",
	}
}

func (g *GitlabHandler) Add(w http.ResponseWriter, r *http.Request, u *User) error {
	settings := database.SettingsMust()
	teams, err := database.ListTeams(u.ID)
	if err != nil {
		return err
	}
	data := struct {
		User     *User
		Teams    []*Team
		Settings *Settings
	}{u, teams, settings}
	// if the user hasn't linked their GitLab account
	// render a different template
	if len(u.GitlabToken) == 0 {
		return RenderTemplate(w, "gitlab_link.html", &data)
	}
	// otherwise display the template for adding
	// a new GitLab repository.
	return RenderTemplate(w, "gitlab_add.html", &data)
}

func (g *GitlabHandler) Link(w http.ResponseWriter, r *http.Request, u *User) error {
	token := strings.TrimSpace(r.FormValue("token"))

	if len(u.GitlabToken) == 0 || token != u.GitlabToken && len(token) > 0 {
		u.GitlabToken = token
		settings := database.SettingsMust()
		gl := gogitlab.NewGitlab(settings.GitlabApiUrl, g.apiPath, u.GitlabToken)
		_, err := gl.CurrentUser()
		if err != nil {
			return fmt.Errorf("Private Token is not valid: %q", err)
		}
		if err := database.SaveUser(u); err != nil {
			return RenderError(w, err, http.StatusBadRequest)
		}
	}

	http.Redirect(w, r, "/new/gitlab", http.StatusSeeOther)
	return nil
}

func (g *GitlabHandler) ReLink(w http.ResponseWriter, r *http.Request, u *User) error {
	data := struct {
		User *User
	}{u}
	return RenderTemplate(w, "gitlab_link.html", &data)
}

func (g *GitlabHandler) Create(w http.ResponseWriter, r *http.Request, u *User) error {
	teamName := r.FormValue("team")
	owner := r.FormValue("owner")
	name := r.FormValue("name")

	repo, err := g.newGitlabRepo(u, owner, name)
	if err != nil {
		return err
	}

	if len(teamName) > 0 {
		team, err := database.GetTeamSlug(teamName)
		if err != nil {
			return fmt.Errorf("Unable to find Team %s.", teamName)
		}

		// user must be an admin member of the team
		if ok, _ := database.IsMemberAdmin(u.ID, team.ID); !ok {
			return fmt.Errorf("Invalid permission to access Team %s.", teamName)
		}
		repo.TeamID = team.ID
	}

	// Save to the database
	if err := database.SaveRepo(repo); err != nil {
		return fmt.Errorf("Error saving repository to the database. %s", err)
	}

	return RenderText(w, http.StatusText(http.StatusOK), http.StatusOK)
}

func (g *GitlabHandler) newGitlabRepo(u *User, owner, name string) (*Repo, error) {
	settings := database.SettingsMust()
	gl := gogitlab.NewGitlab(settings.GitlabApiUrl, g.apiPath, u.GitlabToken)

	project, err := gl.Project(ns(owner, name))
	if err != nil {
		return nil, err
	}

	var cloneUrl string
	if project.Public {
		cloneUrl = project.HttpRepoUrl
	} else {
		cloneUrl = project.SshRepoUrl
	}

	repo, err := NewRepo(settings.GitlabDomain, owner, name, ScmGit, cloneUrl)
	if err != nil {
		return nil, err
	}

	repo.UserID = u.ID
	repo.Private = !project.Public
	if repo.Private {
		// name the key
		keyName := fmt.Sprintf("%s@%s", repo.Owner, settings.Domain)

		// TODO: (fudanchii) check if we already opted to use UserKey

		// create the github key, or update if one already exists
		if err := gl.AddProjectDeployKey(ns(owner, name), keyName, repo.PublicKey); err != nil {
			return nil, fmt.Errorf("Unable to add Public Key to your GitLab repository.")
		}
	}

	link := fmt.Sprintf("%s://%s/hook/gitlab?id=%s", settings.Scheme, settings.Domain, repo.Slug)
	if err := gl.AddProjectHook(ns(owner, name), link, true, false, true); err != nil {
		return nil, fmt.Errorf("Unable to add Hook to your GitLab repository.")
	}

	return repo, err
}

func (g *GitlabHandler) Hook(w http.ResponseWriter, r *http.Request) error {
	rID := r.FormValue("id")
	repo, err := database.GetRepoSlug(rID)
	if err != nil {
		return RenderText(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}

	user, err := database.GetUser(repo.UserID)
	if err != nil {
		return RenderText(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}

	payload, _ := ioutil.ReadAll(r.Body)
	parsed, err := gogitlab.ParseHook(payload)
	if err != nil {
		return err
	}
	if parsed.ObjectKind == "merge_request" {
		fmt.Println(string(payload))
		if err := g.PullRequestHook(parsed, repo, user); err != nil {
			return err
		}
		return RenderText(w, http.StatusText(http.StatusOK), http.StatusOK)
	}

	if len(parsed.After) == 0 {
		return RenderText(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}

	_, err = database.GetCommitHash(parsed.After, repo.ID)
	if err != nil && err != sql.ErrNoRows {
		fmt.Println("commit already exists")
		return RenderText(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
	}

	commit := &Commit{}
	commit.RepoID = repo.ID
	commit.Branch = parsed.Branch()
	commit.Hash = parsed.After
	commit.Status = "Pending"
	commit.Created = time.Now().UTC()

	head := parsed.Head()
	commit.Message = head.Message
	commit.Timestamp = head.Timestamp
	if head.Author != nil {
		commit.SetAuthor(head.Author.Email)
	} else {
		commit.SetAuthor(parsed.UserName)
	}

	// get the github settings from the database
	settings := database.SettingsMust()

	// get the drone.yml file from GitHub
	client := gogitlab.NewGitlab(settings.GitlabApiUrl, g.apiPath, user.GitlabToken)

	buildscript, err := client.RepoRawFile(ns(repo.Owner, repo.Name), commit.Hash, ".drone.yml")
	if err != nil {
		msg := "No .drone.yml was found in this repository.  You need to add one.\n"
		if err := saveFailedBuild(commit, msg); err != nil {
			return RenderText(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return RenderText(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	}

	// save the commit to the database
	if err := database.SaveCommit(commit); err != nil {
		return RenderText(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	// save the build to the database
	build := &Build{}
	build.Slug = "1" // TODO
	build.CommitID = commit.ID
	build.Created = time.Now().UTC()
	build.Status = "Pending"
	build.BuildScript = string(buildscript)
	if err := database.SaveBuild(build); err != nil {
		return RenderText(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	g.queue.Add(&queue.BuildTask{Repo: repo, Commit: commit, Build: build})

	// OK!
	return RenderText(w, http.StatusText(http.StatusOK), http.StatusOK)

}

func (g *GitlabHandler) PullRequestHook(p *gogitlab.HookPayload, repo *Repo, user *User) error {
	obj := p.ObjectAttributes

	// Gitlab may trigger multiple hooks upon updating merge requests status
	// only build when it was just opened and the merge hasn't been checked yet.
	if !(obj.State == "opened" && obj.MergeStatus == "unchecked") {
		fmt.Println("Ignore GitLab Merge Requests")
		return nil
	}

	settings := database.SettingsMust()

	client := gogitlab.NewGitlab(settings.GitlabApiUrl, g.apiPath, user.GitlabToken)

	// GitLab merge-requests hook doesn't include repository data.
	// Have to fetch it manually
	src, err := client.RepoBranch(strconv.Itoa(obj.SourceProjectId), obj.SourceBranch)
	if err != nil {
		return err
	}

	_, err = database.GetCommitHash(src.Commit.Id, repo.ID)
	if err != nil && err != sql.ErrNoRows {
		fmt.Println("commit already exists")
		return err
	}

	commit := &Commit{}
	commit.RepoID = repo.ID
	commit.Branch = src.Name
	commit.Hash = src.Commit.Id
	commit.Status = "Pending"
	commit.Created = time.Now().UTC()
	commit.PullRequest = strconv.Itoa(obj.IId)

	commit.Message = src.Commit.Message
	commit.Timestamp = src.Commit.AuthoredDateRaw
	commit.SetAuthor(src.Commit.Author.Email)

	buildscript, err := client.RepoRawFile(strconv.Itoa(obj.SourceProjectId), commit.Hash, ".drone.yml")
	if err != nil {
		msg := "No .drone.yml was found in this repository.  You need to add one.\n"
		if err := saveFailedBuild(commit, msg); err != nil {
			return fmt.Errorf("Failed to save build: %q", err)
		}
		return fmt.Errorf("Error to fetch build script: %q", err)
	}

	// save the commit to the database
	if err := database.SaveCommit(commit); err != nil {
		return fmt.Errorf("Failed to save commit: %q", err)
	}

	// save the build to the database
	build := &Build{}
	build.Slug = "1" // TODO
	build.CommitID = commit.ID
	build.Created = time.Now().UTC()
	build.Status = "Pending"
	build.BuildScript = string(buildscript)
	if err := database.SaveBuild(build); err != nil {
		return fmt.Errorf("Failed to save build: %q", err)
	}

	g.queue.Add(&queue.BuildTask{Repo: repo, Commit: commit, Build: build})

	return nil
}

// ns namespaces user and repo.
// Returns user%2Frepo
func ns(user, repo string) string {
	return fmt.Sprintf("%s%%2F%s", user, repo)
}
