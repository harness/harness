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
	} `json:"sender"`

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

type deployHook struct {
	Deployment struct {
		ID   int64  `json:"id"`
		Sha  string `json:"sha"`
		Ref  string `json:"ref"`
		Task string `json:"task"`
		Env  string `json:"environment"`
		Url  string `json:"url"`
		Desc string `json:"description"`
	} `json:"deployment"`

	Sender struct {
		Login  string `json:"login"`
		Avatar string `json:"avatar_url"`
	} `json:"sender"`

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

	// these are legacy fields that have been moded
	// to the deployment section. They are here for
	// older versions of GitHub and will be removed

	ID   int64  `json:"id"`
	Sha  string `json:"sha"`
	Ref  string `json:"ref"`
	Name string `json:"name"`
	Env  string `json:"environment"`
	Desc string `json:"description"`
}
