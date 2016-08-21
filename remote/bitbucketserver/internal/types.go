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

type CloneLink struct {
	Href string `json:"href"`
	Name string `json:"name"`
}

type SelfRefLink struct {
	Href string `json:"href"`
}

type BuildStatus struct {
	State string `json:"state"`
	Key   string `json:"key"`
	Name  string `json:"name,omitempty"`
	Url   string `json:"url"`
	Desc  string `json:"description,omitempty"`
}

type Repo struct {
	Forkable bool `json:"forkable"`
	ID       int  `json:"id"`
	Links    struct {
		Clone []CloneLink `json:"clone"`
		Self  []struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"links"`
	Name    string `json:"name"`
	Project struct {
		Description string `json:"description"`
		ID          int    `json:"id"`
		Key         string `json:"key"`
		Links       struct {
			Self []SelfRefLink `json:"self"`
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
	IsLastPage bool    `json:"isLastPage"`
	Limit      int     `json:"limit"`
	Size       int     `json:"size"`
	Start      int     `json:"start"`
	Values     []*Repo `json:"values"`
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

type Value struct {
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
}

type PostHook struct {
	Changesets struct {
		Filter     interface{} `json:"filter"`
		IsLastPage bool        `json:"isLastPage"`
		Limit      int         `json:"limit"`
		Size       int         `json:"size"`
		Start      int         `json:"start"`
		Values     []Value     `json:"values"`
	} `json:"changesets"`
	RefChanges []RefChange `json:"refChanges"`
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

type RefChange struct {
	FromHash string `json:"fromHash"`
	RefID    string `json:"refId"`
	ToHash   string `json:"toHash"`
	Type     string `json:"type"`
}
