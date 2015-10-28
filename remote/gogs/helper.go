package gogs

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/drone/drone/model"
	"github.com/gogits/go-gogs-client"
)

// helper function that converts a Gogs repository
// to a Drone repository.
func toRepoLite(from *gogs.Repository) *model.RepoLite {
	name := strings.Split(from.FullName, "/")[1]
	avatar := expandAvatar(
		from.HtmlUrl,
		from.Owner.AvatarUrl,
	)
	return &model.RepoLite{
		Name:     name,
		Owner:    from.Owner.UserName,
		FullName: from.FullName,
		Avatar:   avatar,
	}
}

// helper function that converts a Gogs repository
// to a Drone repository.
func toRepo(from *gogs.Repository) *model.Repo {
	name := strings.Split(from.FullName, "/")[1]
	avatar := expandAvatar(
		from.HtmlUrl,
		from.Owner.AvatarUrl,
	)
	return &model.Repo{
		Kind:      model.RepoGit,
		Name:      name,
		Owner:     from.Owner.UserName,
		FullName:  from.FullName,
		Avatar:    avatar,
		Link:      from.HtmlUrl,
		IsPrivate: from.Private,
		Clone:     from.CloneUrl,
		Branch:    "master",
	}
}

// helper function that converts a Gogs permission
// to a Drone permission.
func toPerm(from gogs.Permission) *model.Perm {
	return &model.Perm{
		Pull:  from.Pull,
		Push:  from.Push,
		Admin: from.Admin,
	}
}

// helper function that extracts the Build data
// from a Gogs push hook
func buildFromPush(hook *PushHook) *model.Build {
	avatar := expandAvatar(
		hook.Repo.Url,
		fixMalformedAvatar(hook.Sender.Avatar),
	)
	return &model.Build{
		Event:     model.EventPush,
		Commit:    hook.After,
		Ref:       hook.Ref,
		Link:      hook.Compare,
		Branch:    strings.TrimPrefix(hook.Ref, "refs/heads/"),
		Message:   hook.Commits[0].Message,
		Avatar:    avatar,
		Author:    hook.Sender.Login,
		Timestamp: time.Now().UTC().Unix(),
	}
}

// helper function that extracts the Repository data
// from a Gogs push hook
func repoFromPush(hook *PushHook) *model.Repo {
	fullName := fmt.Sprintf(
		"%s/%s",
		hook.Repo.Owner.Username,
		hook.Repo.Name,
	)
	return &model.Repo{
		Name:     hook.Repo.Name,
		Owner:    hook.Repo.Owner.Username,
		FullName: fullName,
		Link:     hook.Repo.Url,
	}
}

// helper function that parses a push hook from
// a read closer.
func parsePush(r io.Reader) (*PushHook, error) {
	push := new(PushHook)
	err := json.NewDecoder(r).Decode(push)
	return push, err
}

// fixMalformedAvatar is a helper function that fixes
// an avatar url if malformed (known bug with gogs)
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

// expandAvatar is a helper function that converts
// a relative avatar URL to the abosolute url.
func expandAvatar(repo, rawurl string) string {
	if !strings.HasPrefix(rawurl, "/avatars/") {
		return rawurl
	}
	url_, err := url.Parse(repo)
	if err != nil {
		return rawurl
	}
	url_.Path = rawurl
	return url_.String()
}
