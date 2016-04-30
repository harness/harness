package internal

import (
	"net/url"
	"strconv"
	"time"
)

type Account struct {
	Login string `json:"username"`
	Name  string `json:"display_name"`
	Type  string `json:"type"`
	Links Links  `json:"links"`
}

type AccountResp struct {
	Page   int        `json:"page"`
	Pages  int        `json:"pagelen"`
	Size   int        `json:"size"`
	Next   string     `json:"next"`
	Values []*Account `json:"values"`
}

type BuildStatus struct {
	State string `json:"state"`
	Key   string `json:"key"`
	Name  string `json:"name,omitempty"`
	Url   string `json:"url"`
	Desc  string `json:"description,omitempty"`
}

type Email struct {
	Email       string `json:"email"`
	IsConfirmed bool   `json:"is_confirmed"`
	IsPrimary   bool   `json:"is_primary"`
}

type EmailResp struct {
	Page   int      `json:"page"`
	Pages  int      `json:"pagelen"`
	Size   int      `json:"size"`
	Next   string   `json:"next"`
	Values []*Email `json:"values"`
}

type Hook struct {
	Uuid   string   `json:"uuid,omitempty"`
	Desc   string   `json:"description"`
	Url    string   `json:"url"`
	Events []string `json:"events"`
	Active bool     `json:"active"`
}

type HookResp struct {
	Page   int     `json:"page"`
	Pages  int     `json:"pagelen"`
	Size   int     `json:"size"`
	Next   string  `json:"next"`
	Values []*Hook `json:"values"`
}

type Links struct {
	Avatar Link   `json:"avatar"`
	Html   Link   `json:"html"`
	Clone  []Link `json:"clone"`
}

type Link struct {
	Href string `json:"href"`
	Name string `json:"name"`
}

type LinkClone struct {
	Link
}

type Repo struct {
	Owner     Account `json:"owner"`
	Name      string  `json:"name"`
	FullName  string  `json:"full_name"`
	Language  string  `json:"language"`
	IsPrivate bool    `json:"is_private"`
	Scm       string  `json:"scm"`
	Desc      string  `json:"desc"`
	Links     Links   `json:"links"`
}

type RepoResp struct {
	Page   int     `json:"page"`
	Pages  int     `json:"pagelen"`
	Size   int     `json:"size"`
	Next   string  `json:"next"`
	Values []*Repo `json:"values"`
}

type Source struct {
	Node string `json:"node"`
	Path string `json:"path"`
	Data string `json:"data"`
	Size int64  `json:"size"`
}

type Change struct {
	New struct {
		Type   string `json:"type"`
		Name   string `json:"name"`
		Target struct {
			Type    string    `json:"type"`
			Hash    string    `json:"hash"`
			Message string    `json:"message"`
			Date    time.Time `json:"date"`
			Links   Links     `json:"links"`
			Author  struct {
				Raw  string  `json:"raw"`
				User Account `json:"user"`
			} `json:"author"`
		} `json:"target"`
	} `json:"new"`
}

type PushHook struct {
	Actor Account `json:"actor"`
	Repo  Repo    `json:"repository"`
	Push  struct {
		Changes []Change `json:"changes"`
	} `json:"push"`
}

type PullRequestHook struct {
	Actor       Account `json:"actor"`
	Repo        Repo    `json:"repository"`
	PullRequest struct {
		ID      int       `json:"id"`
		Type    string    `json:"type"`
		Reason  string    `json:"reason"`
		Desc    string    `json:"description"`
		Title   string    `json:"title"`
		State   string    `json:"state"`
		Links   Links     `json:"links"`
		Created time.Time `json:"created_on"`
		Updated time.Time `json:"updated_on"`

		Source struct {
			Repo   Repo `json:"repsoitory"`
			Commit struct {
				Hash  string `json:"hash"`
				Links Links  `json:"links"`
			} `json:"commit"`
			Branch struct {
				Name string `json:"name"`
			} `json:"branch"`
		} `json:"source"`

		Dest struct {
			Repo   Repo `json:"repsoitory"`
			Commit struct {
				Hash  string `json:"hash"`
				Links Links  `json:"links"`
			} `json:"commit"`
			Branch struct {
				Name string `json:"name"`
			} `json:"branch"`
		} `json:"destination"`
	} `json:"pullrequest"`
}

type ListOpts struct {
	Page    int
	PageLen int
}

func (o *ListOpts) Encode() string {
	params := url.Values{}
	if o.Page != 0 {
		params.Set("page", strconv.Itoa(o.Page))
	}
	if o.PageLen != 0 {
		params.Set("pagelen", strconv.Itoa(o.PageLen))
	}
	return params.Encode()
}

type ListTeamOpts struct {
	Page    int
	PageLen int
	Role    string
}

func (o *ListTeamOpts) Encode() string {
	params := url.Values{}
	if o.Page != 0 {
		params.Set("page", strconv.Itoa(o.Page))
	}
	if o.PageLen != 0 {
		params.Set("pagelen", strconv.Itoa(o.PageLen))
	}
	if len(o.Role) != 0 {
		params.Set("role", o.Role)
	}
	return params.Encode()
}

type Error struct {
	Status int
	Body   struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (e Error) Error() string {
	return e.Body.Message
}
