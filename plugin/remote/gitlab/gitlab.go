package gitlab

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"code.google.com/p/goauth2/oauth"
	"github.com/Bugagazavr/go-gitlab-client"
	"github.com/drone/drone/shared/httputil"
	"github.com/drone/drone/shared/model"
)

type Gitlab struct {
	url        string
	SkipVerify bool
	Open       bool
	Client     string
	Secret     string
}

func New(url string, skipVerify, open bool, client, secret string) *Gitlab {
	return &Gitlab{
		url:        url,
		SkipVerify: skipVerify,
		Open:       open,
		Client:     client,
		Secret:     secret,
	}
}

// Authorize handles authentication with thrid party remote systems,
// such as github or bitbucket, and returns user data.
func (r *Gitlab) Authorize(res http.ResponseWriter, req *http.Request) (*model.Login, error) {
	host := httputil.GetURL(req)
	config := NewOauthConfig(r, host)

	var code = req.FormValue("code")
	var state = req.FormValue("state")

	if len(code) == 0 {
		var random = GetRandom()
		httputil.SetCookie(res, req, "gitlab_state", random)
		http.Redirect(res, req, config.AuthCodeURL(random), http.StatusSeeOther)
		return nil, nil
	}

	cookieState := httputil.GetCookie(req, "gitlab_state")
	httputil.DelCookie(res, req, "gitlab_state")
	if cookieState != state {
		return nil, fmt.Errorf("Error matching state in OAuth2 redirect")
	}

	var trans = &oauth.Transport{Config: config}
	var token, err = trans.Exchange(code)
	if err != nil {
		return nil, fmt.Errorf("Error exchanging token. %s", err)
	}

	var client = NewClient(r.url, token.AccessToken, r.SkipVerify)

	var user, errr = client.CurrentUser()
	if errr != nil {
		return nil, fmt.Errorf("Error retrieving current user. %s", errr)
	}

	var login = new(model.Login)
	login.ID = int64(user.Id)
	login.Access = token.AccessToken
	login.Secret = token.RefreshToken
	login.Login = user.Username
	login.Email = user.Email
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

func (r *Gitlab) GetToken(user *model.User) (*model.Token, error) {
	expiry := time.Unix(user.TokenExpiry, 0)
	if expiry.Sub(time.Now()) > (60 * time.Second) {
		return nil, nil
	}

	t := &oauth.Transport{
		Config: NewOauthConfig(r, ""),
		Token: &oauth.Token{
			AccessToken:  user.Access,
			RefreshToken: user.Secret,
			Expiry:       expiry,
		},
	}

	if err := t.Refresh(); err != nil {
		return nil, err
	}

	var token = new(model.Token)
	token.AccessToken = t.Token.AccessToken
	token.RefreshToken = t.Token.RefreshToken
	token.Expiry = t.Token.Expiry.Unix()
	return token, nil
}
