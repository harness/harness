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

// helper function that converts a Gogs repository to a Drone repository.
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

// helper function that converts a Gogs repository to a Drone repository.
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

// helper function that converts a Gogs permission to a Drone permission.
func toPerm(from gogs.Permission) *model.Perm {
	return &model.Perm{
		Pull:  from.Pull,
		Push:  from.Push,
		Admin: from.Admin,
	}
}

// helper function that converts a Gogs team to a Drone team.
func toTeam(from *gogs.Organization, link string) *model.Team {
	return &model.Team{
		Login:  from.UserName,
		Avatar: expandAvatar(link, from.AvatarUrl),
	}
}

// helper function that extracts the Build data from a Gogs push hook
func buildFromPush(hook *pushHook) *model.Build {
	avatar := expandAvatar(
		hook.Repo.URL,
		fixMalformedAvatar(hook.Sender.Avatar),
	)

	var eventType string
	var message string

	switch {
	case hook.RefType == "tag":
		eventType = model.EventTag
		message = "Tag " + hook.Ref
	default:
		eventType = model.EventPush
		message = hook.Commits[0].Message
	}

	return &model.Build{
		Event:     eventType,
		Commit:    hook.After,
		Ref:       hook.Ref,
		Link:      hook.Compare,
		Branch:    strings.TrimPrefix(hook.Ref, "refs/heads/"),
		Message:   message,
		Avatar:    avatar,
		Author:    hook.Sender.Login,
		Timestamp: time.Now().UTC().Unix(),
	}
}

// helper function that extracts the Repository data from a Gogs push hook
func repoFromPush(hook *pushHook) *model.Repo {
	fullName := fmt.Sprintf(
		"%s/%s",
		hook.Repo.Owner.Username,
		hook.Repo.Name,
	)
	return &model.Repo{
		Name:     hook.Repo.Name,
		Owner:    hook.Repo.Owner.Username,
		FullName: fullName,
		Link:     hook.Repo.URL,
	}
}

// helper function that parses a push hook from a read closer.
func parsePush(r io.Reader) (*pushHook, error) {
	push := new(pushHook)
	err := json.NewDecoder(r).Decode(push)
	return push, err
}

// fixMalformedAvatar is a helper function that fixes an avatar url if malformed
// (currently a known bug with gogs)
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
