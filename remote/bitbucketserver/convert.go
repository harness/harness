package bitbucketserver

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote/bitbucketserver/internal"
	"github.com/mrjones/oauth"
)

const (
	statusPending = "INPROGRESS"
	statusSuccess = "SUCCESSFUL"
	statusFailure = "FAILED"
)

const (
	descPending = "this build is pending"
	descSuccess = "the build was successful"
	descFailure = "the build failed"
	descError   = "oops, something went wrong"
)

// convertStatus is a helper function used to convert a Drone status to a
// Bitbucket commit status.
func convertStatus(status string) string {
	switch status {
	case model.StatusPending, model.StatusRunning:
		return statusPending
	case model.StatusSuccess:
		return statusSuccess
	default:
		return statusFailure
	}
}

// convertDesc is a helper function used to convert a Drone status to a
// Bitbucket status description.
func convertDesc(status string) string {
	switch status {
	case model.StatusPending, model.StatusRunning:
		return descPending
	case model.StatusSuccess:
		return descSuccess
	case model.StatusFailure:
		return descFailure
	default:
		return descError
	}
}

// convertRepo is a helper function used to convert a Bitbucket server repository
// structure to the common Drone repository structure.
func convertRepo(from *internal.Repo) *model.Repo {

	repo := model.Repo{
		Name:      from.Slug,
		Owner:     from.Project.Key,
		Branch:    "master",
		Kind:      model.RepoGit,
		IsPrivate: true, // Since we have to use Netrc it has to always be private :/
		FullName:  fmt.Sprintf("%s/%s", from.Project.Key, from.Slug),
	}

	for _, item := range from.Links.Clone {
		if item.Name == "http" {
			uri, err := url.Parse(item.Href)
			if err != nil {
				return nil
			}
			uri.User = nil
			repo.Clone = uri.String()
		}
	}
	for _, item := range from.Links.Self {
		if item.Href != "" {
			repo.Link = item.Href
		}
	}
	return &repo

}

// convertRepoLite is a helper function used to convert a Bitbucket repository
// structure to the simplified Drone repository structure.
func convertRepoLite(from *internal.Repo) *model.RepoLite {
	return &model.RepoLite{
		Owner:    from.Project.Key,
		Name:     from.Slug,
		FullName: from.Project.Key + "/" + from.Slug,
		//TODO: find the avatar for the repo
		//Avatar: might need another ws call?
	}

}

// convertPushHook is a helper function used to convert a Bitbucket push
// hook to the Drone build struct holding commit information.
func convertPushHook(hook *internal.PostHook, baseURL string) *model.Build {
	branch := strings.TrimPrefix(
		strings.TrimPrefix(
			hook.RefChanges[0].RefID,
			"refs/heads/",
		),
		"refs/tags/",
	)

	//Ensuring the author label is not longer then 40 for the label of the commit author (default size in the db)
	authorLabel := hook.Changesets.Values[0].ToCommit.Author.Name
	if len(authorLabel) > 40 {
		authorLabel = authorLabel[0:37] + "..."
	}

	build := &model.Build{
		Commit:    hook.RefChanges[0].ToHash, // TODO check for index value
		Branch:    branch,
		Message:   hook.Changesets.Values[0].ToCommit.Message, //TODO check for index Values
		Avatar:    avatarLink(hook.Changesets.Values[0].ToCommit.Author.EmailAddress),
		Author:    authorLabel,
		Email:     hook.Changesets.Values[0].ToCommit.Author.EmailAddress,
		Timestamp: time.Now().UTC().Unix(),
		Ref:       hook.RefChanges[0].RefID, // TODO check for index Values
		Link:      fmt.Sprintf("%s/projects/%s/repos/%s/commits/%s", baseURL, hook.Repository.Project.Key, hook.Repository.Slug, hook.RefChanges[0].ToHash),
	}
	if strings.HasPrefix(hook.RefChanges[0].RefID, "refs/tags/") {
		build.Event = model.EventTag
	} else {
		build.Event = model.EventPush
	}

	return build
}

// convertUser is a helper function used to convert a Bitbucket user account
// structure to the Drone User structure.
func convertUser(from *internal.User, token *oauth.AccessToken) *model.User {
	return &model.User{
		Login:  from.Slug,
		Token:  token.Token,
		Email:  from.EmailAddress,
		Avatar: avatarLink(from.EmailAddress),
	}
}

func avatarLink(email string) string {
	hasher := md5.New()
	hasher.Write([]byte(strings.ToLower(email)))
	emailHash := fmt.Sprintf("%v", hex.EncodeToString(hasher.Sum(nil)))
	avatarURL := fmt.Sprintf("https://www.gravatar.com/avatar/%s.jpg", emailHash)
	return avatarURL
}
