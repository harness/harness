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

package github

type webhook struct {
	Ref     string `json:"ref"`
	Action  string `json:"action"`
	Deleted bool   `json:"deleted"`
	BaseRef string `json:"base_ref"`

	Head struct {
		ID        string `json:"id"`
		URL       string `json:"url"`
		Message   string `json:"message"`
		Timestamp string `json:"timestamp"`

		Author struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Username string `json:"username"`
		} `json:"author"`

		Committer struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Username string `json:"username"`
		} `json:"committer"`
	} `json:"head_commit"`

	Sender struct {
		Login  string `json:"login"`
		Avatar string `json:"avatar_url"`
	} `json:"sender"`

	// repository details
	Repo struct {
		Owner struct {
			Login string `json:"login"`
			Name  string `json:"name"`
		} `json:"owner"`

		Name          string `json:"name"`
		FullName      string `json:"full_name"`
		Language      string `json:"language"`
		Private       bool   `json:"private"`
		HTMLURL       string `json:"html_url"`
		CloneURL      string `json:"clone_url"`
		DefaultBranch string `json:"default_branch"`
	} `json:"repository"`

	// deployment hook details
	Deployment struct {
		ID   int64  `json:"id"`
		Sha  string `json:"sha"`
		Ref  string `json:"ref"`
		Task string `json:"task"`
		Env  string `json:"environment"`
		URL  string `json:"url"`
		Desc string `json:"description"`
	} `json:"deployment"`

	// pull request details
	PullRequest struct {
		Number  int    `json:"number"`
		State   string `json:"state"`
		Title   string `json:"title"`
		HTMLURL string `json:"html_url"`

		User struct {
			Login  string `json:"login"`
			Avatar string `json:"avatar_url"`
		} `json:"user"`

		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`

		Head struct {
			SHA  string `json:"sha"`
			Ref  string `json:"ref"`
			Repo struct {
				CloneURL string `json:"clone_url"`
			} `json:"repo"`
		} `json:"head"`
	} `json:"pull_request"`
}
