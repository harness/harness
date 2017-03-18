package github

import (
	"fmt"
	"strings"

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
	descPending  = "this build is pending"
	descSuccess  = "the build was successful"
	descFailure  = "the build failed"
	descBlocked  = "the build requires approval"
	descDeclined = "the build was rejected"
	descError    = "oops, something went wrong"
)

const (
	headRefs  = "refs/pull/%d/head"  // pull request unmerged
	mergeRefs = "refs/pull/%d/merge" // pull request merged with base
	refspec   = "%s:%s"
)

// convertStatus is a helper function used to convert a Drone status to a
// GitHub commit status.
func convertStatus(status string) string {
	switch status {
	case model.StatusPending, model.StatusRunning, model.StatusBlocked:
		return statusPending
	case model.StatusFailure, model.StatusDeclined:
		return statusFailure
	case model.StatusSuccess:
		return statusSuccess
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
	case model.StatusBlocked:
		return descBlocked
	case model.StatusDeclined:
		return descDeclined
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

// convertTeamPerm is a helper function used to convert a GitHub organization
// permissions to the common Drone permissions structure.
func convertTeamPerm(from *github.Membership) *model.Perm {
	admin := false
	if *from.Role == "admin" {
		admin = true
	}
	return &model.Perm{
		Admin: admin,
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

// convertRepoHook is a helper function used to extract the Repository details
// from a webhook and convert to the common Drone repository structure.
func convertRepoHook(from *webhook) *model.Repo {
	repo := &model.Repo{
		Owner:     from.Repo.Owner.Login,
		Name:      from.Repo.Name,
		FullName:  from.Repo.FullName,
		Link:      from.Repo.HTMLURL,
		IsPrivate: from.Repo.Private,
		Clone:     from.Repo.CloneURL,
		Branch:    from.Repo.DefaultBranch,
		Kind:      model.RepoGit,
	}
	if repo.Branch == "" {
		repo.Branch = defaultBranch
	}
	if repo.Owner == "" { // legacy webhooks
		repo.Owner = from.Repo.Owner.Name
	}
	if repo.FullName == "" {
		repo.FullName = repo.Owner + "/" + repo.Name
	}
	return repo
}

// convertPushHook is a helper function used to extract the Build details
// from a push webhook and convert to the common Drone Build structure.
func convertPushHook(from *webhook) *model.Build {
	build := &model.Build{
		Event:   model.EventPush,
		Commit:  from.Head.ID,
		Ref:     from.Ref,
		Link:    from.Head.URL,
		Branch:  strings.Replace(from.Ref, "refs/heads/", "", -1),
		Message: from.Head.Message,
		Email:   from.Head.Author.Email,
		Avatar:  from.Sender.Avatar,
		Author:  from.Sender.Login,
		Remote:  from.Repo.CloneURL,
		Sender:  from.Sender.Login,
	}
	if len(build.Author) == 0 {
		build.Author = from.Head.Author.Username
	}
	if len(build.Email) == 0 {
		// default to gravatar?
	}
	if strings.HasPrefix(build.Ref, "refs/tags/") {
		// just kidding, this is actually a tag event. Why did this come as a push
		// event we'll never know!
		build.Event = model.EventTag
	}
	return build
}

// convertPushHook is a helper function used to extract the Build details
// from a deploy webhook and convert to the common Drone Build structure.
func convertDeployHook(from *webhook) *model.Build {
	build := &model.Build{
		Event:   model.EventDeploy,
		Commit:  from.Deployment.Sha,
		Link:    from.Deployment.URL,
		Message: from.Deployment.Desc,
		Avatar:  from.Sender.Avatar,
		Author:  from.Sender.Login,
		Ref:     from.Deployment.Ref,
		Branch:  from.Deployment.Ref,
		Deploy:  from.Deployment.Env,
		Sender:  from.Sender.Login,
	}
	// if the ref is a sha or short sha we need to manuallyconstruct the ref.
	if strings.HasPrefix(build.Commit, build.Ref) || build.Commit == build.Ref {
		build.Branch = from.Repo.DefaultBranch
		if build.Branch == "" {
			build.Branch = defaultBranch
		}
		build.Ref = fmt.Sprintf("refs/heads/%s", build.Branch)
	}
	// if the ref is a branch we should make sure it has refs/heads prefix
	if !strings.HasPrefix(build.Ref, "refs/") { // branch or tag
		build.Ref = fmt.Sprintf("refs/heads/%s", build.Branch)
	}
	return build
}

// convertPullHook is a helper function used to extract the Build details
// from a pull request webhook and convert to the common Drone Build structure.
func convertPullHook(from *webhook, merge bool) *model.Build {
	build := &model.Build{
		Event:   model.EventPull,
		Commit:  from.PullRequest.Head.SHA,
		Link:    from.PullRequest.HTMLURL,
		Ref:     fmt.Sprintf(headRefs, from.PullRequest.Number),
		Branch:  from.PullRequest.Base.Ref,
		Message: from.PullRequest.Title,
		Author:  from.PullRequest.User.Login,
		Avatar:  from.PullRequest.User.Avatar,
		Title:   from.PullRequest.Title,
		Sender:  from.Sender.Login,
		Remote:  from.PullRequest.Head.Repo.CloneURL,
		Refspec: fmt.Sprintf(refspec,
			from.PullRequest.Head.Ref,
			from.PullRequest.Base.Ref,
		),
	}
	if merge {
		build.Ref = fmt.Sprintf(mergeRefs, from.PullRequest.Number)
	}
	return build
}
