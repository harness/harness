package swagger

import (
	"net/http"

	"github.com/drone/drone/model"
)

// swagger:parameters getUser deleteUser
type loginParam struct {
	// User's login name
	//
	// in: path
	// required: true
	Login string `json:"login"`
}

// swagger:parameters updateRepo getBuild getBuildList getRepo patchBuild deleteBuild
type ownerParam struct {
	// Repository owner
	//
	// in: path
	// required: true
	Owner string `json:"owner"`
}

// swagger:parameters updateRepo getBuild getBuildList getRepo patchBuild deleteBuild
type nameParam struct {
	// Repository name
	//
	// in: path
	// required: true
	Name string `json:"name"`
}

// swagger:parameters getBuild patchBuild deleteBuild
type buildParam struct {
	// Build identifier
	//
	// in: path
	// required: true
	Build string `json:"build"`
}

// Update repository data
//
// swagger:parameters repoPatch
type repoPatchParam struct {
	// in: body
	Body []model.RepoPatch
}

// swagger:route GET /users/{login} user getUser
//
// Get the user with the matching login.
//
//     Responses:
//       200: user
//
func getUser(w http.ResponseWriter, r *http.Request) {}

// swagger:route GET /user user getCurrentUser
//
// Get the currently authenticated user.
//
//     Responses:
//       200: user
//
func userCurrent(w http.ResponseWriter, r *http.Request) {}

// swagger:route GET /users user getUserList
//
// Get the list of all registered users.
//
//     Responses:
//       200: users
//
func userList(w http.ResponseWriter, r *http.Request) {}

// swagger:route POST /user/token user tokenRequest
//
// Get the currently authenticated user's build feed.
//
//     Responses:
//       200: feed
//
func userToken(w http.ResponseWriter, r *http.Request) {}

// swagger:route DELETE /users/{login} user deleteUser
//
// Delete the user with the matching login.
//
//     Responses:
//       200: user
//
func deleteUser(w http.ResponseWriter, r *http.Request) {}

// swagger:route GET /user/repos user getUserRepos
//
// Get the currently authenticated user's active repository list.
//
//     Responses:
//       200: repos
//
func listRepo(w http.ResponseWriter, r *http.Request) {}

// swagger:route GET /repos/{owner}/{name} repository getRepo
//
// Get the specified repository.
//
//     Responses:
//       200: repo
//
func getRepo(w http.ResponseWriter, r *http.Request) {}

// swagger:route PATCH /repos/{owner}/{name} repository updateRepo
//
// Update the specified repository.
//
//     Responses:
//       200: repo
//
func updateRepo(w http.ResponseWriter, r *http.Request) {}

// swagger:route GET /repos/{owner}/{name}/builds build getBuildList
//
// Get list of builds for specified repository.
//
//     Responses:
//       200: builds
//
func listBuilds(w http.ResponseWriter, r *http.Request) {}

// swagger:route GET /repos/{owner}/{name}/builds/{build} build getBuild
//
// Get the specified build.
//
//     Responses:
//       200: build
//
func getBuild(w http.ResponseWriter, r *http.Request) {}

// swagger:route PATCH /repos/{owner}/{name}/builds/{build} build patchBuild
//
// Update the specified build.
//
//     Responses:
//       200: build
//
func patchBuild(w http.ResponseWriter, r *http.Request) {}

// swagger:route DELETE /repos/{owner}/{name}/builds/{build} build deleteBuild
//
// Delete the specified build.
//
//     Responses:
//       200: build
//
func deleteBuild(w http.ResponseWriter, r *http.Request) {}

// An user
//
// swagger:response user
type userResp struct {
	// in: body
	Body model.User
}

// A collection of users
//
// swagger:response users
type usersResp struct {
	// in: body
	Body []model.User
}

// A feed entry for a build.
//
// Feed entries can be used to display information on the latest builds.
//
// swagger:response feed
type feedResp struct {
	// in: body
	Body []model.Feed
}

// A collection of repositories
//
// swagger:response repos
type reposResp struct {
	// in: body
	Body []model.Repo
}

// A build
//
// swagger:response build
type buildResp struct {
	// in: body
	Body model.Build
}

// A build list
//
// swagger:response builds
type buildsResp struct {
	// in: body
	Body []model.Build
}

// A process response logs
//
// swagger:response logs
type procLogsResp struct {
	// in: body
	Body []model.Proc
}
