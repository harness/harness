package bitbucket

type Account struct {
	Login string `json:"username"`
	Name  string `json:"display_name"`
	Type  string `json:"type"`
	Links Links  `json:"links"`
}

type AccountResp struct {
	Page   int       `json:"page"`
	Pages  int       `json:"pagelen"`
	Size   int       `json:"size"`
	Values []Account `json:"values"`
}

type Email struct {
	Email       string `json:"email"`
	IsConfirmed bool   `json:"is_confirmed"`
	IsPrimary   bool   `json:"is_primary"`
}

type EmailResp struct {
	Page   int     `json:"page"`
	Pages  int     `json:"pagelen"`
	Size   int     `json:"size"`
	Values []Email `json:"values"`
}

type Hook struct {
	Uuid   string   `json:"uuid,omitempty"`
	Desc   string   `json:"description"`
	Url    string   `json:"url"`
	Events []string `json:"events"`
	Active bool     `json:"active"`
}

type HookResp struct {
	Page   int    `json:"page"`
	Pages  int    `json:"pagelen"`
	Size   int    `json:"size"`
	Values []Hook `json:"values"`
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
	Page   int    `json:"page"`
	Pages  int    `json:"pagelen"`
	Size   int    `json:"size"`
	Values []Repo `json:"values"`
}

type ListOpts struct {
	Page    int
	PageLen int
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
