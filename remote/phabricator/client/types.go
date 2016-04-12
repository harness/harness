package client

type QMap map[string]string

type User struct {
	Id         string   `json:"phid,omitempty"`
	Name       string   `json:"realName,omitempty"`
	Username   string   `json:"userName,omitempty"`
	Email      string   `json:"primaryEmail,omitempty"`
	AvatarUrl  string   `json:"image,omitempty"`
	ProfileUrl string   `json:"uri,omitempty"`
	Roles      []string `json:"roles,omitempty"`
}
