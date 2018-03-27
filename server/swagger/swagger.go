// Copyright 2018 Drone.IO Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package swagger

import (
	"net/http"

	"github.com/drone/drone/model"
)

// swagger:route GET /users/{login} user getUser
//
// Get the user with the matching login.
//
//     Responses:
//       200: user
//
func userFind(w http.ResponseWriter, r *http.Request) {}

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
//       200: user
//
func userList(w http.ResponseWriter, r *http.Request) {}

// swagger:route GET /user/feed user getUserFeed
//
// Get the currently authenticated user's build feed.
//
//     Responses:
//       200: feed
//
func userFeed(w http.ResponseWriter, r *http.Request) {}

// swagger:route DELETE /users/{login} user deleteUserLogin
//
// Delete the user with the matching login.
//
//     Responses:
//       200: user
//
func userDelete(w http.ResponseWriter, r *http.Request) {}

// swagger:route GET /user/repos user getUserRepos
//
// Get the currently authenticated user's active repository list.
//
//     Responses:
//       200: repos
//
func repoList(w http.ResponseWriter, r *http.Request) {}

// swagger:response user
type userResp struct {
	// in: body
	Body model.User
}

// swagger:response users
type usersResp struct {
	// in: body
	Body []model.User
}

// swagger:response feed
type feedResp struct {
	// in: body
	Body []model.Feed
}

// swagger:response repos
type reposResp struct {
	// in: body
	Body []model.Repo
}
