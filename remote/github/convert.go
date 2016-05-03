package github

import (
	"github.com/drone/drone/model"

	"github.com/google/go-github/github"
)

const defaultBranch = "master"

const (
	statusPending = "pending"
	statusSuccess = "success"
	statusFailure = "failure"
	statusError   = "error"
)

const (
	descPending = "this build is pending"
	descSuccess = "the build was successful"
	descFailure = "the build failed"
	descError   = "oops, something went wrong"
)

// convertStatus is a helper function used to convert a Drone status to a
// GitHub commit status.
func convertStatus(status string) string {
	switch status {
	case model.StatusPending, model.StatusRunning:
		return statusPending
	case model.StatusSuccess:
		return statusSuccess
	case model.StatusFailure:
		return statusFailure
	default:
		return statusError
	}
}

// convertDesc is a helper function used to convert a Drone status to a
// GitHub status description.
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

// convertRepo is a helper function used to convert a GitHub repository
// structure to the common Drone repository structure.
func convertRepo(from *github.Repository, private bool) *model.Repo {
	repo := &model.Repo{
		Owner:     *from.Owner.Login,
		Name:      *from.Name,
		FullName:  *from.FullName,
		Link:      *from.HTMLURL,
		IsPrivate: *from.Private,
		Clone:     *from.CloneURL,
		Avatar:    *from.Owner.AvatarURL,
		Kind:      model.RepoGit,
		Branch:    defaultBranch,
	}
	if from.DefaultBranch != nil {
		repo.Branch = *from.DefaultBranch
	}
	if private {
		repo.IsPrivate = true
	}
	return repo
}

// convertPerm is a helper function used to convert a GitHub repository
// permissions to the common Drone permissions structure.
func convertPerm(from *github.Repository) *model.Perm {
	return &model.Perm{
		Admin: (*from.Permissions)["admin"],
		Push:  (*from.Permissions)["push"],
		Pull:  (*from.Permissions)["pull"],
	}
}

// convertRepoList is a helper function used to convert a GitHub repository
// list to the common Drone repository structure.
func convertRepoList(from []github.Repository) []*model.RepoLite {
	var repos []*model.RepoLite
	for _, repo := range from {
		repos = append(repos, convertRepoLite(repo))
	}
	return repos
}

// convertRepoLite is a helper function used to convert a GitHub repository
// structure to the common Drone repository structure.
func convertRepoLite(from github.Repository) *model.RepoLite {
	return &model.RepoLite{
		Owner:    *from.Owner.Login,
		Name:     *from.Name,
		FullName: *from.FullName,
		Avatar:   *from.Owner.AvatarURL,
	}
}

// convertTeamList is a helper function used to convert a GitHub team list to
// the common Drone repository structure.
func convertTeamList(from []github.Organization) []*model.Team {
	var teams []*model.Team
	for _, team := range from {
		teams = append(teams, convertTeam(team))
	}
	return teams
}

// convertTeam is a helper function used to convert a GitHub team structure
// to the common Drone repository structure.
func convertTeam(from github.Organization) *model.Team {
	return &model.Team{
		Login:  *from.Login,
		Avatar: *from.AvatarURL,
	}
}
