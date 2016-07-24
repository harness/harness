package bitbucketserver

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/drone/drone/model"
	"github.com/drone/drone/remote/bitbucketserver/internal"
	"net/url"
	"strings"
	"github.com/mrjones/oauth"
)

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
	log.Debug(fmt.Printf("Repo: %+v\n", repo))
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
func convertPushHook(hook *internal.PostHook) *model.Build {
	build := &model.Build{
		Commit: hook.RefChanges[0].ToHash, // TODO check for index value
		//Link: TODO find link
		Branch: strings.Split(hook.RefChanges[0].RefID, "refs/heads/")[1], //TODO figure the correct for tags
		Message: hook.Changesets.Values[0].ToCommit.Message, //TODO check for index Values
		Avatar: avatarLink(hook.Changesets.Values[0].ToCommit.Author.EmailAddress),
		Author: hook.Changesets.Values[0].ToCommit.Author.EmailAddress, // TODO check for index Values
		//Timestamp: TODO find time parsing
		Event: model.EventPush,          //TODO: do more then PUSH find Tags etc
		Ref:   hook.RefChanges[0].RefID, // TODO check for index Values

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
	log.Debug(avatarURL)
	return avatarURL
}
