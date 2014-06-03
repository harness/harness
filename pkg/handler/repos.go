package handler

import (
	"fmt"
	"net/http"

	"github.com/drone/drone/pkg/channel"
	"github.com/drone/drone/pkg/database"
	. "github.com/drone/drone/pkg/model"
	"github.com/drone/go-bitbucket/bitbucket"
	"github.com/drone/go-github/github"

	"launchpad.net/goyaml"
)

// Display a Repository dashboard.
func RepoDashboard(w http.ResponseWriter, r *http.Request, u *User, repo *Repo) error {
	branch := r.FormValue("branch")

	// get a list of all branches
	branches, err := database.ListBranches(repo.ID)
	if err != nil {
		return err
	}

	// if no branch is provided then we'll
	// want to use a default value.
	if len(branch) == 0 {
		branch = repo.DefaultBranch()
	}

	// get a list of recent commits for the
	// repository and specific branch
	commits, err := database.ListCommits(repo.ID, branch)
	if err != nil {
		return err
	}

	// get a token that can be exchanged with the
	// websocket handler to authorize listening
	// for a stream of changes for this repository
	token := channel.Create(repo.Slug)

	data := struct {
		User     *User
		Repo     *Repo
		Branches []*Commit
		Commits  []*Commit
		Branch   string
		Token    string
	}{u, repo, branches, commits, branch, token}

	return RenderTemplate(w, "repo_dashboard.html", &data)
}

func RepoAddGithub(w http.ResponseWriter, r *http.Request, u *User) error {
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
	// if the user hasn't linked their GitHub account
	// render a different template
	if len(u.GithubToken) == 0 {
		return RenderTemplate(w, "github_link.html", &data)
	}
	// otherwise display the template for adding
	// a new GitHub repository.
	return RenderTemplate(w, "github_add.html", &data)
}

func RepoAddBitbucket(w http.ResponseWriter, r *http.Request, u *User) error {
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
	// if the user hasn't linked their Bitbucket account
	// render a different template
	if len(u.BitbucketToken) == 0 {
		return RenderTemplate(w, "bitbucket_link.html", &data)
	}
	// otherwise display the template for adding
	// a new Bitbucket repository.
	return RenderTemplate(w, "bitbucket_add.html", &data)
}

func RepoCreateGithub(w http.ResponseWriter, r *http.Request, u *User) error {
	teamName := r.FormValue("team")
	owner := r.FormValue("owner")
	name := r.FormValue("name")

	// get the github settings from the database
	settings := database.SettingsMust()

	// create the GitHub client
	client := github.New(u.GithubToken)
	client.ApiUrl = settings.GitHubApiUrl
	githubRepo, err := client.Repos.Find(owner, name)
	if err != nil {
		return fmt.Errorf("Unable to find GitHub repository %s/%s.", owner, name)
	}

	repo, err := NewGitHubRepo(settings.GitHubDomain, owner, name, githubRepo.Private)
	if err != nil {
		return err
	}

	repo.UserID = u.ID
	repo.Private = githubRepo.Private

	// if the user chose to assign to a team account
	// we need to retrieve the team, verify the user
	// has access, and then set the team id.
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

	// if the repository is private we'll need
	// to upload a github key to the repository
	if repo.Private {
		// name the key
		keyName := fmt.Sprintf("%s@%s", repo.Owner, settings.Domain)

		// create the github key, or update if one already exists
		_, err := client.RepoKeys.CreateUpdate(owner, name, repo.PublicKey, keyName)
		if err != nil {
			return fmt.Errorf("Unable to add Public Key to your GitHub repository.")
		}
	} else {

	}

	// create a hook so that we get notified when code
	// is pushed to the repository and can execute a build.
	link := fmt.Sprintf("%s://%s/hook/github.com?id=%s", settings.Scheme, settings.Domain, repo.Slug)

	// add the hook
	if _, err := client.Hooks.CreateUpdate(owner, name, link); err != nil {
		return fmt.Errorf("Unable to add Hook to your GitHub repository.")
	}

	// Save to the database
	if err := database.SaveRepo(repo); err != nil {
		return fmt.Errorf("Error saving repository to the database. %s", err)
	}

	return RenderText(w, http.StatusText(http.StatusOK), http.StatusOK)
}

func RepoCreateBitbucket(w http.ResponseWriter, r *http.Request, u *User) error {
	teamName := r.FormValue("team")
	owner := r.FormValue("owner")
	name := r.FormValue("name")

	// get the bitbucket settings from the database
	settings := database.SettingsMust()

	// create the Bitbucket client
	client := bitbucket.New(
		settings.BitbucketKey,
		settings.BitbucketSecret,
		u.BitbucketToken,
		u.BitbucketSecret,
	)

	bitbucketRepo, err := client.Repos.Find(owner, name)
	if err != nil {
		return fmt.Errorf("Unable to find Bitbucket repository %s/%s.", owner, name)
	}

	repo, err := NewBitbucketRepo(owner, name, bitbucketRepo.Private)
	if err != nil {
		return err
	}

	repo.UserID = u.ID
	repo.Private = bitbucketRepo.Private

	// if the user chose to assign to a team account
	// we need to retrieve the team, verify the user
	// has access, and then set the team id.
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

	// if the repository is private we'll need
	// to upload a bitbucket key to the repository
	if repo.Private {
		// name the key
		keyName := fmt.Sprintf("%s@%s", repo.Owner, settings.Domain)

		// create the bitbucket key, or update if one already exists
		_, err := client.RepoKeys.CreateUpdate(owner, name, repo.PublicKey, keyName)
		if err != nil {
			return fmt.Errorf("Unable to add Public Key to your Bitbucket repository: %s", err)
		}
	} else {

	}

	// create a hook so that we get notified when code
	// is pushed to the repository and can execute a build.
	link := fmt.Sprintf("%s://%s/hook/bitbucket.org?id=%s", settings.Scheme, settings.Domain, repo.Slug)

	// add the hook
	if _, err := client.Brokers.CreateUpdate(owner, name, link, bitbucket.BrokerTypePost); err != nil {
		return fmt.Errorf("Unable to add Hook to your Bitbucket repository. %s", err.Error())
	}

	// Save to the database
	if err := database.SaveRepo(repo); err != nil {
		return fmt.Errorf("Error saving repository to the database. %s", err)
	}

	return RenderText(w, http.StatusText(http.StatusOK), http.StatusOK)
}

// Repository Settings
func RepoSettingsForm(w http.ResponseWriter, r *http.Request, u *User, repo *Repo) error {

	// get the list of teams
	teams, err := database.ListTeams(u.ID)
	if err != nil {
		return err
	}

	data := struct {
		Repo  *Repo
		User  *User
		Teams []*Team
		Owner *User
		Team  *Team
	}{Repo: repo, User: u, Teams: teams}

	// get the repo owner
	if repo.TeamID > 0 {
		data.Team, err = database.GetTeam(repo.TeamID)
		if err != nil {
			return err
		}
	}

	// get the team owner
	data.Owner, err = database.GetUser(repo.UserID)
	if err != nil {
		return err
	}

	return RenderTemplate(w, "repo_settings.html", &data)
}

// Repository Params (YAML parameters) Form
func RepoParamsForm(w http.ResponseWriter, r *http.Request, u *User, repo *Repo) error {

	data := struct {
		Repo     *Repo
		User     *User
		Textarea string
	}{repo, u, ""}

	if repo.Params != nil && len(repo.Params) != 0 {
		raw, _ := goyaml.Marshal(&repo.Params)
		data.Textarea = string(raw)
	}

	return RenderTemplate(w, "repo_params.html", &data)
}

func RepoBadges(w http.ResponseWriter, r *http.Request, u *User, repo *Repo) error {
	// hostname from settings
	hostname := database.SettingsMust().URL().String()

	data := struct {
		Repo *Repo
		User *User
		Host string
	}{repo, u, hostname}
	return RenderTemplate(w, "repo_badges.html", &data)
}

func RepoKeys(w http.ResponseWriter, r *http.Request, u *User, repo *Repo) error {
	data := struct {
		Repo *Repo
		User *User
	}{repo, u}
	return RenderTemplate(w, "repo_keys.html", &data)
}

// Updates an existing repository.
func RepoUpdate(w http.ResponseWriter, r *http.Request, u *User, repo *Repo) error {
	switch r.FormValue("action") {
	case "params":
		repo.Params = map[string]string{}
		if err := goyaml.Unmarshal([]byte(r.FormValue("params")), &repo.Params); err != nil {
			return err
		}
	default:
		repo.URL = r.FormValue("URL")
		repo.Disabled = len(r.FormValue("Disabled")) == 0
		repo.DisabledPullRequest = len(r.FormValue("DisabledPullRequest")) == 0
		repo.Private = len(r.FormValue("Private")) > 0
		repo.Privileged = u.Admin && len(r.FormValue("Privileged")) > 0

		// value of "" indicates the currently authenticated user
		// should be set as the administrator.
		if len(r.FormValue("Owner")) == 0 {
			repo.UserID = u.ID
			repo.TeamID = 0
		} else {
			// else the user has chosen a team
			team, err := database.GetTeamSlug(r.FormValue("Owner"))
			if err != nil {
				return err
			}

			// verify the user is a member of the team
			if member, _ := database.IsMemberAdmin(u.ID, team.ID); !member {
				return fmt.Errorf("Forbidden")
			}

			// set the team ID
			repo.TeamID = team.ID
		}
	}

	// save the page
	if err := database.SaveRepo(repo); err != nil {
		return err
	}

	http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)
	return nil
}

// Deletes a specific repository.
func RepoDeleteForm(w http.ResponseWriter, r *http.Request, u *User, repo *Repo) error {
	data := struct {
		Repo *Repo
		User *User
	}{repo, u}
	return RenderTemplate(w, "repo_delete.html", &data)
}

// Deletes a specific repository.
func RepoDelete(w http.ResponseWriter, r *http.Request, u *User, repo *Repo) error {
	// the user must confirm their password before deleting
	password := r.FormValue("password")
	if err := u.ComparePassword(password); err != nil {
		return RenderError(w, err, http.StatusBadRequest)
	}

	// delete the repo
	if err := database.DeleteRepo(repo.ID); err != nil {
		return err
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	return nil
}
