package github

type postHook struct {
}

type pushHook struct {
	Ref     string `json:"ref"`
	Deleted bool   `json:"deleted"`

	Head struct {
		ID        string `json:"id"`
		URL       string `json:"url"`
		Message   string `json:"message"`
		Timestamp string `json:"timestamp"`

		Author struct {
			Name     string `json:"name"`
			Email    string `json:"name"`
			Username string `json:"username"`
		} `json:"author"`

		Committer struct {
			Name     string `json:"name"`
			Email    string `json:"name"`
			Username string `json:"username"`
		} `json:"committer"`
	} `json:"head_commit"`

	Sender struct {
		Login  string `json:"login"`
		Avatar string `json:"avatar_url"`
	}

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
}
