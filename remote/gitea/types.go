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

package gitea

type pushHook struct {
	Sha     string `json:"sha"`
	Ref     string `json:"ref"`
	Before  string `json:"before"`
	After   string `json:"after"`
	Compare string `json:"compare_url"`
	RefType string `json:"ref_type"`

	Pusher struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Login    string `json:"login"`
		Username string `json:"username"`
	} `json:"pusher"`

	Repo struct {
		ID       int64  `json:"id"`
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		URL      string `json:"html_url"`
		Private  bool   `json:"private"`
		Owner    struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Username string `json:"username"`
		} `json:"owner"`
	} `json:"repository"`

	Commits []struct {
		ID      string `json:"id"`
		Message string `json:"message"`
		URL     string `json:"url"`
	} `json:"commits"`

	Sender struct {
		ID       int64  `json:"id"`
		Login    string `json:"login"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Avatar   string `json:"avatar_url"`
	} `json:"sender"`
}

type pullRequestHook struct {
	Action      string `json:"action"`
	Number      int64  `json:"number"`
	PullRequest struct {
		ID   int64 `json:"id"`
		User struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
			Name     string `json:"full_name"`
			Email    string `json:"email"`
			Avatar   string `json:"avatar_url"`
		} `json:"user"`
		Title     string   `json:"title"`
		Body      string   `json:"body"`
		State     string   `json:"state"`
		URL       string   `json:"html_url"`
		Mergeable bool     `json:"mergeable"`
		Merged    bool     `json:"merged"`
		MergeBase string   `json:"merge_base"`
		Base      struct {
			Label string `json:"label"`
			Ref   string `json:"ref"`
			Sha   string `json:"sha"`
			Repo  struct {
				ID       int64  `json:"id"`
				Name     string `json:"name"`
				FullName string `json:"full_name"`
				URL      string `json:"html_url"`
				Private  bool   `json:"private"`
				Owner    struct {
					ID       int64  `json:"id"`
					Username string `json:"username"`
					Name     string `json:"full_name"`
					Email    string `json:"email"`
					Avatar   string `json:"avatar_url"`
				} `json:"owner"`
			} `json:"repo"`
		} `json:"base"`
		Head struct {
			Label string `json:"label"`
			Ref   string `json:"ref"`
			Sha   string `json:"sha"`
			Repo  struct {
				ID       int64  `json:"id"`
				Name     string `json:"name"`
				FullName string `json:"full_name"`
				URL      string `json:"html_url"`
				Private  bool   `json:"private"`
				Owner    struct {
					ID       int64  `json:"id"`
					Username string `json:"username"`
					Name     string `json:"full_name"`
					Email    string `json:"email"`
					Avatar   string `json:"avatar_url"`
				} `json:"owner"`
			} `json:"repo"`
		} `json:"head"`
	} `json:"pull_request"`
	Repo struct {
		ID       int64  `json:"id"`
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		URL      string `json:"html_url"`
		Private  bool   `json:"private"`
		Owner    struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
			Name     string `json:"full_name"`
			Email    string `json:"email"`
			Avatar   string `json:"avatar_url"`
		} `json:"owner"`
	} `json:"repository"`
	Sender struct {
		ID       int64  `json:"id"`
		Login    string `json:"login"`
		Username string `json:"username"`
		Name     string `json:"full_name"`
		Email    string `json:"email"`
		Avatar   string `json:"avatar_url"`
	} `json:"sender"`
}
