package gogitlab

import (
	"encoding/json"
	"net/url"
)

const (
	session_path = "/session"
)

type Session struct {
	Id               int    `json:"id"`
	UserName         string `json:"username"`
	Name             string `json:"name"`
	Blocked          bool   `json:"blocked"`
	State            string `json:"state"`
	AvatarURL        string `json:"avatar_url",omitempty`
	IsAdmin          bool   `json:"is_admin"`
	Bio              string `json:"bio",omitempty`
	Email            string `json:"email"`
	ThemeId          int    `json:"theme_id",omitempty`
	ColorSchemeId    int    `json:"color_scheme_id",omitempty`
	ExternUid        string `json:"extern_uid",omitempty`
	Provider         string `json:"provider",omitempty`
	CanCreateGroup   bool   `json:"can_create_group"`
	CanCreateProject bool   `json:"can_create_project"`
	Skype            string `json:"skype",omitempty`
	Twitter          string `json:"twitter",omitempty`
	LinkedIn         string `json:"linkedin",omitempty`
	WebsiteURL       string `json:"website_url",omitempty`
	PrivateToken     string `json:"private_token"`
}

func (g *Gitlab) GetSession(email string, password string) (*Session, error) {
	session_url := g.ResourceUrl(session_path, map[string]string{})

	var session *Session

	v := url.Values{}
	v.Set("email", email)
	v.Set("password", password)

	body := v.Encode()

	contents, err := g.buildAndExecRequest("POST", session_url, []byte(body))
	if err != nil {
		return session, err
	}

	err = json.Unmarshal(contents, &session)

	return session, err
}
