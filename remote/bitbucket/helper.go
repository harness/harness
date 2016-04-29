package bitbucket

import (
	"net/url"
	"strings"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote/bitbucket/internal"

	"golang.org/x/oauth2"
)

// convertRepo is a helper function used to convert a Bitbucket repository
// structure to the common Drone repository structure.
func convertRepo(from *internal.Repo) *model.Repo {
	repo := model.Repo{
		Clone:     cloneLink(from),
		Owner:     strings.Split(from.FullName, "/")[0],
		Name:      strings.Split(from.FullName, "/")[1],
		FullName:  from.FullName,
		Link:      from.Links.Html.Href,
		IsPrivate: from.IsPrivate,
		Avatar:    from.Owner.Links.Avatar.Href,
		Kind:      from.Scm,
		Branch:    "master",
	}
	if repo.Kind == model.RepoHg {
		repo.Branch = "default"
	}
	return &repo
}

// cloneLink is a helper function that tries to extract the clone url from the
// repository object.
func cloneLink(repo *internal.Repo) string {
	var clone string

	// above we manually constructed the repository clone url. below we will
	// iterate through the list of clone links and attempt to instead use the
	// clone url provided by bitbucket.
	for _, link := range repo.Links.Clone {
		if link.Name == "https" {
			clone = link.Href
		}
	}

	// if no repository name is provided, we use the Html link. this excludes the
	// .git suffix, but will still clone the repo.
	if len(clone) == 0 {
		clone = repo.Links.Html.Href
	}

	// if bitbucket tries to automatically populate the user in the url we must
	// strip it out.
	cloneurl, err := url.Parse(clone)
	if err == nil {
		cloneurl.User = nil
		clone = cloneurl.String()
	}

	return clone
}

// convertRepoLite is a helper function used to convert a Bitbucket repository
// structure to the simplified Drone repository structure.
func convertRepoLite(from *internal.Repo) *model.RepoLite {
	return &model.RepoLite{
		Owner:    strings.Split(from.FullName, "/")[0],
		Name:     strings.Split(from.FullName, "/")[1],
		FullName: from.FullName,
		Avatar:   from.Owner.Links.Avatar.Href,
	}
}

// convertUser is a helper function used to convert a Bitbucket user account
// structure to the Drone User structure.
func convertUser(from *internal.Account, token *oauth2.Token) *model.User {
	return &model.User{
		Login:  from.Login,
		Token:  token.AccessToken,
		Secret: token.RefreshToken,
		Expiry: token.Expiry.UTC().Unix(),
		Avatar: from.Links.Avatar.Href,
	}
}

// convertTeamList is a helper function used to convert a Bitbucket team list
// structure to the Drone Team structure.
func convertTeamList(from []*internal.Account) []*model.Team {
	var teams []*model.Team
	for _, team := range from {
		teams = append(teams, convertTeam(team))
	}
	return teams
}

// convertTeam is a helper function used to convert a Bitbucket team account
// structure to the Drone Team structure.
func convertTeam(from *internal.Account) *model.Team {
	return &model.Team{
		Login:  from.Login,
		Avatar: from.Links.Avatar.Href,
	}
}
