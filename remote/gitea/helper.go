package gitea

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/drone/drone/model"
)

// helper function that converts a Gitea repository to a Drone repository.
func toRepoLite(from *gitea.Repository) *model.RepoLite {
	name := strings.Split(from.FullName, "/")[1]
	avatar := expandAvatar(
		from.HTMLURL,
		from.Owner.AvatarURL,
	)
	return &model.RepoLite{
		Name:     name,
		Owner:    from.Owner.UserName,
		FullName: from.FullName,
		Avatar:   avatar,
	}
}

// helper function that converts a Gitea repository to a Drone repository.
func toRepo(from *gitea.Repository) *model.Repo {
	name := strings.Split(from.FullName, "/")[1]
	avatar := expandAvatar(
		from.HTMLURL,
		from.Owner.AvatarURL,
	)
	return &model.Repo{
		Kind:      model.RepoGit,
		Name:      name,
		Owner:     from.Owner.UserName,
		FullName:  from.FullName,
		Avatar:    avatar,
		Link:      from.HTMLURL,
		IsPrivate: from.Private,
		Clone:     from.CloneURL,
		Branch:    "master",
	}
}

// helper function that converts a Gitea permission to a Drone permission.
func toPerm(from *gitea.Permission) *model.Perm {
	return &model.Perm{
		Pull:  from.Pull,
		Push:  from.Push,
		Admin: from.Admin,
	}
}

// helper function that converts a Gitea team to a Drone team.
func toTeam(from *gitea.Organization, link string) *model.Team {
	return &model.Team{
		Login:  from.UserName,
		Avatar: expandAvatar(link, from.AvatarURL),
	}
}

// helper function that extracts the Build data from a Gitea push hook
func buildFromPush(hook *pushHook) *model.Build {
	avatar := expandAvatar(
		hook.Repo.URL,
		fixMalformedAvatar(hook.Sender.Avatar),
	)
	author := hook.Sender.Login
	if author == "" {
		author = hook.Sender.Username
	}
	sender := hook.Sender.Username
	if sender == "" {
		sender = hook.Sender.Login
	}

	return &model.Build{
		Event:     model.EventPush,
		Commit:    hook.After,
		Ref:       hook.Ref,
		Link:      hook.Compare,
		Branch:    strings.TrimPrefix(hook.Ref, "refs/heads/"),
		Message:   hook.Commits[0].Message,
		Avatar:    avatar,
		Author:    author,
		Email:     hook.Pusher.Email,
		Timestamp: time.Now().UTC().Unix(),
		Sender:    sender,
	}
}

// helper function that extracts the Build data from a Gitea tag hook
func buildFromTag(hook *pushHook) *model.Build {
	avatar := expandAvatar(
		hook.Repo.URL,
		fixMalformedAvatar(hook.Sender.Avatar),
	)
	author := hook.Sender.Login
	if author == "" {
		author = hook.Sender.Username
	}
	sender := hook.Sender.Username
	if sender == "" {
		sender = hook.Sender.Login
	}

	return &model.Build{
		Event:     model.EventTag,
		Commit:    hook.After,
		Ref:       fmt.Sprintf("refs/tags/%s", hook.Ref),
		Link:      fmt.Sprintf("%s/src/%s", hook.Repo.URL, hook.Ref),
		Branch:    fmt.Sprintf("refs/tags/%s", hook.Ref),
		Message:   fmt.Sprintf("created tag %s", hook.Ref),
		Avatar:    avatar,
		Author:    author,
		Sender:    sender,
		Timestamp: time.Now().UTC().Unix(),
	}
}

// helper function that extracts the Build data from a Gitea pull_request hook
func buildFromPullRequest(hook *pullRequestHook) *model.Build {
	avatar := expandAvatar(
		hook.Repo.URL,
		fixMalformedAvatar(hook.PullRequest.User.Avatar),
	)
	sender := hook.Sender.Username
	if sender == "" {
		sender = hook.Sender.Login
	}
	build := &model.Build{
		Event:   model.EventPull,
		Commit:  hook.PullRequest.Head.Sha,
		Link:    hook.PullRequest.URL,
		Ref:     fmt.Sprintf("refs/pull/%d/head", hook.Number),
		Branch:  hook.PullRequest.BaseBranch,
		Message: hook.PullRequest.Title,
		Author:  hook.PullRequest.User.Username,
		Avatar:  avatar,
		Sender:  sender,
		Title:   hook.PullRequest.Title,
		Refspec: fmt.Sprintf("%s:%s",
			hook.PullRequest.HeadBranch,
			hook.PullRequest.BaseBranch,
		),
	}
	return build
}

// helper function that extracts the Repository data from a Gitea push hook
func repoFromPush(hook *pushHook) *model.Repo {
	return &model.Repo{
		Name:     hook.Repo.Name,
		Owner:    hook.Repo.Owner.Username,
		FullName: hook.Repo.FullName,
		Link:     hook.Repo.URL,
	}
}

// helper function that extracts the Repository data from a Gitea pull_request hook
func repoFromPullRequest(hook *pullRequestHook) *model.Repo {
	return &model.Repo{
		Name:     hook.Repo.Name,
		Owner:    hook.Repo.Owner.Username,
		FullName: hook.Repo.FullName,
		Link:     hook.Repo.URL,
	}
}

// helper function that parses a push hook from a read closer.
func parsePush(r io.Reader) (*pushHook, error) {
	push := new(pushHook)
	err := json.NewDecoder(r).Decode(push)
	return push, err
}

func parsePullRequest(r io.Reader) (*pullRequestHook, error) {
	pr := new(pullRequestHook)
	err := json.NewDecoder(r).Decode(pr)
	return pr, err
}

// fixMalformedAvatar is a helper function that fixes an avatar url if malformed
// (currently a known bug with gitea)
func fixMalformedAvatar(url string) string {
	index := strings.Index(url, "///")
	if index != -1 {
		return url[index+1:]
	}
	index = strings.Index(url, "//avatars/")
	if index != -1 {
		return strings.Replace(url, "//avatars/", "/avatars/", -1)
	}
	return url
}

// expandAvatar is a helper function that converts a relative avatar URL to the
// absolute url.
func expandAvatar(repo, rawurl string) string {
	aurl, err := url.Parse(rawurl)
	if err != nil {
		return rawurl
	}
	if aurl.IsAbs() {
		// Url is already absolute
		return aurl.String()
	}

	// Resolve to base
	burl, err := url.Parse(repo)
	if err != nil {
		return rawurl
	}
	aurl = burl.ResolveReference(aurl)

	return aurl.String()
}
