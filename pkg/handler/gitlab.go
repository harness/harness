package handler

import (
	"fmt"
	"net/http"

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
	token := r.FormValue("token")
	u.GitlabToken = token

	if err := database.SaveUser(u); err != nil {
		return RenderError(w, err, http.StatusBadRequest)
	}

	settings := database.SettingsMust()
	gl := gogitlab.NewGitlab(settings.GitlabApiUrl, g.apiPath, u.GitlabToken)
	_, err := gl.CurrentUser()
	if err != nil {
		return fmt.Errorf("Private Token is not valid: %q", err)
	}

	http.Redirect(w, r, "/new/gitlab", http.StatusSeeOther)
	return nil
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

// ns namespaces user and repo.
// Returns user%2Frepo
func ns(user, repo string) string {
	return fmt.Sprintf("%s%%2F%s", user, repo)
}
