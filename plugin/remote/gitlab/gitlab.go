package gitlab

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	"github.com/Bugagazavr/go-gitlab-client"
	"github.com/drone/drone/shared/model"
)

type Gitlab struct {
	url        string
	SkipVerify bool
	Open       bool
}

func New(url string, skipVerify, open bool) *Gitlab {
	return &Gitlab{
		url:        url,
		SkipVerify: skipVerify,
		Open:       open,
	}
}

// Authorize handles authentication with thrid party remote systems,
// such as github or bitbucket, and returns user data.
func (r *Gitlab) Authorize(res http.ResponseWriter, req *http.Request) (*model.Login, error) {
	var username = req.FormValue("username")
	var password = req.FormValue("password")

	var client = NewClient(r.url, "", r.SkipVerify)
	var session, err = client.GetSession(username, password)
	if err != nil {
		return nil, err
	}

	var login = new(model.Login)
	login.ID = int64(session.Id)
	login.Access = session.PrivateToken
	login.Login = session.UserName
	login.Name = session.Name
	login.Email = session.Email
	return login, nil
}

// GetKind returns the identifier of this remote GitHub instane.
func (r *Gitlab) GetKind() string {
	return model.RemoteGitlab
}

// GetHost returns the hostname of this remote GitHub instance.
func (r *Gitlab) GetHost() string {
	uri, _ := url.Parse(r.url)
	return uri.Host
}

// GetRepos fetches all repositories that the specified
// user has access to in the remote system.
func (r *Gitlab) GetRepos(user *model.User) ([]*model.Repo, error) {

	var repos []*model.Repo
	var client = NewClient(r.url, user.Access, r.SkipVerify)
	var list, err = client.AllProjects()
	if err != nil {
		return nil, err
	}

	var remote = r.GetKind()
	var hostname = r.GetHost()

	for _, item := range list {
		var repo = model.Repo{
			UserID:   user.ID,
			Remote:   remote,
			Host:     hostname,
			Owner:    item.Namespace.Path,
			Name:     item.Path,
			Scm:      model.Git,
			Private:  !item.Public,
			CloneURL: item.HttpRepoUrl,
			GitURL:   item.HttpRepoUrl,
			SSHURL:   item.SshRepoUrl,
			URL:      item.Url,
			Role:     &model.Perm{},
		}

		if repo.Private {
			repo.CloneURL = repo.SSHURL
		}

		// if the user is the owner we can assume full access,
		// otherwise check for the permission items.
		if repo.Owner == user.Login {
			repo.Role = new(model.Perm)
			repo.Role.Admin = true
			repo.Role.Write = true
			repo.Role.Read = true
		} else {
			// Fetch current project
			project, err := client.Project(strconv.Itoa(item.Id))
			if err != nil || project.Permissions == nil {
				continue
			}
			repo.Role.Admin = IsAdmin(project)
			repo.Role.Write = IsWrite(project)
			repo.Role.Read = IsRead(project)
		}

		repos = append(repos, &repo)
	}

	return repos, err
}

// GetScript fetches the build script (.drone.yml) from the remote
// repository and returns in string format.
func (r *Gitlab) GetScript(user *model.User, repo *model.Repo, hook *model.Hook) ([]byte, error) {
	var client = NewClient(r.url, user.Access, r.SkipVerify)
	var path = ns(repo.Owner, repo.Name)
	return client.RepoRawFile(path, hook.Sha, ".drone.yml")
}

// Activate activates a repository by adding a Post-commit hook and
// a Public Deploy key, if applicable.
func (r *Gitlab) Activate(user *model.User, repo *model.Repo, link string) error {
	var client = NewClient(r.url, user.Access, r.SkipVerify)
	var path = ns(repo.Owner, repo.Name)
	var title, err = GetKeyTitle(link)
	if err != nil {
		return err
	}

	// if the repository is private we'll need
	// to upload a github key to the repository
	if repo.Private {
		var err = client.AddProjectDeployKey(path, title, repo.PublicKey)
		if err != nil {
			return err
		}
	}

	// append the repo owner / name to the hook url since gitlab
	// doesn't send this detail in the post-commit hook
	link += "?owner=" + repo.Owner + "&name=" + repo.Name

	// add the hook
	return client.AddProjectHook(path, link, true, false, true)
}

// ParseHook parses the post-commit hook from the Request body
// and returns the required data in a standard format.
func (r *Gitlab) ParseHook(req *http.Request) (*model.Hook, error) {

	defer req.Body.Close()
	var payload, _ = ioutil.ReadAll(req.Body)
	var parsed, err = gogitlab.ParseHook(payload)
	if err != nil {
		return nil, err
	}

	if len(parsed.After) == 0 || parsed.TotalCommitsCount == 0 {
		return nil, nil
	}

	if parsed.ObjectKind == "merge_request" {
		// TODO (bradrydzewski) figure out how to handle merge requests
		return nil, nil
	}

	if len(parsed.After) == 0 {
		return nil, nil
	}

	var hook = new(model.Hook)
	hook.Owner = req.FormValue("owner")
	hook.Repo = req.FormValue("name")
	hook.Sha = parsed.After
	hook.Branch = parsed.Branch()

	var head = parsed.Head()
	hook.Message = head.Message
	hook.Timestamp = head.Timestamp

	// extracts the commit author (ideally email)
	// from the post-commit hook
	switch {
	case head.Author != nil:
		hook.Author = head.Author.Email
	case head.Author == nil:
		hook.Author = parsed.UserName
	}

	return hook, nil
}

func (r *Gitlab) OpenRegistration() bool {
	return r.Open
}
