package bitbucketserver

type postHook struct {
	Changesets struct {
		Filter     interface{} `json:"filter"`
		IsLastPage bool        `json:"isLastPage"`
		Limit      int         `json:"limit"`
		Size       int         `json:"size"`
		Start      int         `json:"start"`
		Values     []struct {
			Changes struct {
				Filter     interface{} `json:"filter"`
				IsLastPage bool        `json:"isLastPage"`
				Limit      int         `json:"limit"`
				Size       int         `json:"size"`
				Start      int         `json:"start"`
				Values     []struct {
					ContentID  string `json:"contentId"`
					Executable bool   `json:"executable"`
					Link       struct {
						Rel string `json:"rel"`
						URL string `json:"url"`
					} `json:"link"`
					NodeType string `json:"nodeType"`
					Path     struct {
						Components []string `json:"components"`
						Extension  string   `json:"extension"`
						Name       string   `json:"name"`
						Parent     string   `json:"parent"`
						ToString   string   `json:"toString"`
					} `json:"path"`
					PercentUnchanged int    `json:"percentUnchanged"`
					SrcExecutable    bool   `json:"srcExecutable"`
					Type             string `json:"type"`
				} `json:"values"`
			} `json:"changes"`
			FromCommit struct {
				DisplayID string `json:"displayId"`
				ID        string `json:"id"`
			} `json:"fromCommit"`
			Link struct {
				Rel string `json:"rel"`
				URL string `json:"url"`
			} `json:"link"`
			ToCommit struct {
				Author struct {
					EmailAddress string `json:"emailAddress"`
					Name         string `json:"name"`
				} `json:"author"`
				AuthorTimestamp int    `json:"authorTimestamp"`
				DisplayID       string `json:"displayId"`
				ID              string `json:"id"`
				Message         string `json:"message"`
				Parents         []struct {
					DisplayID string `json:"displayId"`
					ID        string `json:"id"`
				} `json:"parents"`
			} `json:"toCommit"`
		} `json:"values"`
	} `json:"changesets"`
	RefChanges []struct {
		FromHash string `json:"fromHash"`
		RefID    string `json:"refId"`
		ToHash   string `json:"toHash"`
		Type     string `json:"type"`
	} `json:"refChanges"`
	Repository struct {
		Forkable bool   `json:"forkable"`
		ID       int    `json:"id"`
		Name     string `json:"name"`
		Project  struct {
			ID         int    `json:"id"`
			IsPersonal bool   `json:"isPersonal"`
			Key        string `json:"key"`
			Name       string `json:"name"`
			Public     bool   `json:"public"`
			Type       string `json:"type"`
		} `json:"project"`
		Public        bool   `json:"public"`
		ScmID         string `json:"scmId"`
		Slug          string `json:"slug"`
		State         string `json:"state"`
		StatusMessage string `json:"statusMessage"`
	} `json:"repository"`
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
