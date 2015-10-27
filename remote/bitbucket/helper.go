package bitbucket

import (
	"net/url"
	"strings"

	"github.com/drone/drone/model"
)

// convertRepo is a helper function used to convert a Bitbucket
// repository structure to the common Drone repository structure.
func convertRepo(from *Repo) *model.Repo {
	repo := model.Repo{
		Owner:     from.Owner.Login,
		Name:      from.Name,
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

	// in some cases, the owner of the repository is not
	// provided, however, we do have the full name.
	if len(repo.Owner) == 0 {
		repo.Owner = strings.Split(repo.FullName, "/")[0]
	}

	// above we manually constructed the repository clone url.
	// below we will iterate through the list of clone links and
	// attempt to instead use the clone url provided by bitbucket.
	for _, link := range from.Links.Clone {
		if link.Name == "https" {
			repo.Clone = link.Href
			break
		}
	}

	// if no repository name is provided, we use the Html link.
	// this excludes the .git suffix, but will still clone the repo.
	if len(repo.Clone) == 0 {
		repo.Clone = repo.Link
	}

	// if bitbucket tries to automatically populate the user
	// in the url we must strip it out.
	clone, err := url.Parse(repo.Clone)
	if err == nil {
		clone.User = nil
		repo.Clone = clone.String()
	}

	return &repo
}

// cloneLink is a helper function that tries to extract the
// clone url from the repository object.
func cloneLink(repo Repo) string {
	var clone string

	// above we manually constructed the repository clone url.
	// below we will iterate through the list of clone links and
	// attempt to instead use the clone url provided by bitbucket.
	for _, link := range repo.Links.Clone {
		if link.Name == "https" {
			clone = link.Href
		}
	}

	// if no repository name is provided, we use the Html link.
	// this excludes the .git suffix, but will still clone the repo.
	if len(clone) == 0 {
		clone = repo.Links.Html.Href
	}

	// if bitbucket tries to automatically populate the user
	// in the url we must strip it out.
	cloneurl, err := url.Parse(clone)
	if err == nil {
		cloneurl.User = nil
		clone = cloneurl.String()
	}

	return clone
}

// convertRepoLite is a helper function used to convert a Bitbucket
// repository structure to the simplified Drone repository structure.
func convertRepoLite(from *Repo) *model.RepoLite {
	return &model.RepoLite{
		Owner:    from.Owner.Login,
		Name:     from.Name,
		FullName: from.FullName,
		Avatar:   from.Owner.Links.Avatar.Href,
	}
}
