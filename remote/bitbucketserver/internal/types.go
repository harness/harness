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

type HookPluginDetails struct {
	Details struct {
		Key           string `json:"key"`
		Name          string `json:"name"`
		Type          string `json:"type"`
		Description   string `json:"description"`
		Version       string `json:"version"`
		ConfigFormKey string `json:"configFormKey"`
	} `json:"details"`
	Enabled    bool `json:"enabled"`
	Configured bool `json:"configured"`
}

type HookSettings struct {
	HookURL0  string `json:"hook-url-0,omitempty"`
	HookURL1  string `json:"hook-url-1,omitempty"`
	HookURL2  string `json:"hook-url-2,omitempty"`
	HookURL3  string `json:"hook-url-3,omitempty"`
	HookURL4  string `json:"hook-url-4,omitempty"`
	HookURL5  string `json:"hook-url-5,omitempty"`
	HookURL6  string `json:"hook-url-6,omitempty"`
	HookURL7  string `json:"hook-url-7,omitempty"`
	HookURL8  string `json:"hook-url-8,omitempty"`
	HookURL9  string `json:"hook-url-9,omitempty"`
	HookURL10 string `json:"hook-url-10,omitempty"`
	HookURL11 string `json:"hook-url-11,omitempty"`
	HookURL12 string `json:"hook-url-12,omitempty"`
	HookURL13 string `json:"hook-url-13,omitempty"`
	HookURL14 string `json:"hook-url-14,omitempty"`
	HookURL15 string `json:"hook-url-15,omitempty"`
	HookURL16 string `json:"hook-url-16,omitempty"`
	HookURL17 string `json:"hook-url-17,omitempty"`
	HookURL18 string `json:"hook-url-18,omitempty"`
	HookURL19 string `json:"hook-url-19,omitempty"`
}
