package gogs

type PushHook struct {
	Ref     string `json:"ref"`
	Before  string `json:"before"`
	After   string `json:"after"`
	Compare string `json:"compare_url"`

	Pusher struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Username string `json:"username"`
	} `json:"pusher"`

	Repo struct {
		ID      int64  `json:"id"`
		Name    string `json:"name"`
		Url     string `json:"url"`
		Private bool   `json:"private"`
		Owner   struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Username string `json:"username"`
		} `json:"owner"`
	} `json:"repository"`

	Commits []struct {
		ID      string `json:"id"`
		Message string `json:"message"`
		Url     string `json:"url"`
	} `json:"commits"`

	Sender struct {
		ID     int64  `json:"id"`
		Login  string `json:"login"`
		Avatar string `json:"avatar_url"`
	} `json:"sender"`
}
