package internal

type User struct {
	Active       bool   `json:"active"`
	DisplayName  string `json:"displayName"`
	EmailAddress string `json:"emailAddress"`
	ID           int    `json:"id"`
	Links        struct {
		Self []struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"links"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	Type string `json:"type"`
}

type BSRepo struct {
	Forkable bool `json:"forkable"`
	ID       int  `json:"id"`
	Links    struct {
		Clone []struct {
			Href string `json:"href"`
			Name string `json:"name"`
		} `json:"clone"`
		Self []struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"links"`
	Name    string `json:"name"`
	Project struct {
		Description string `json:"description"`
		ID          int    `json:"id"`
		Key         string `json:"key"`
		Links       struct {
			Self []struct {
				Href string `json:"href"`
			} `json:"self"`
		} `json:"links"`
		Name   string `json:"name"`
		Public bool   `json:"public"`
		Type   string `json:"type"`
	} `json:"project"`
	Public        bool   `json:"public"`
	ScmID         string `json:"scmId"`
	Slug          string `json:"slug"`
	State         string `json:"state"`
	StatusMessage string `json:"statusMessage"`
}

type Repos struct {
	IsLastPage bool `json:"isLastPage"`
	Limit      int  `json:"limit"`
	Size       int  `json:"size"`
	Start      int  `json:"start"`
	Values     []struct {
		Forkable bool `json:"forkable"`
		ID       int  `json:"id"`
		Links    struct {
			Clone []struct {
				Href string `json:"href"`
				Name string `json:"name"`
			} `json:"clone"`
			Self []struct {
				Href string `json:"href"`
			} `json:"self"`
		} `json:"links"`
		Name    string `json:"name"`
		Project struct {
			Description string `json:"description"`
			ID          int    `json:"id"`
			Key         string `json:"key"`
			Links       struct {
				Self []struct {
					Href string `json:"href"`
				} `json:"self"`
			} `json:"links"`
			Name   string `json:"name"`
			Public bool   `json:"public"`
			Type   string `json:"type"`
		} `json:"project"`
		Public        bool   `json:"public"`
		ScmID         string `json:"scmId"`
		Slug          string `json:"slug"`
		State         string `json:"state"`
		StatusMessage string `json:"statusMessage"`
	} `json:"values"`
}

type Hook struct {
	Enabled bool        `json:"enabled"`
	Details *HookDetail `json:"details"`
}

type HookDetail struct {
	Key           string `json:"key"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Description   string `json:"description"`
	Version       string `json:"version"`
	ConfigFormKey string `json:"configFormKey"`
}
