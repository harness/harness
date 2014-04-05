package github

type Org struct {
	Login  string `json:"login"`
	Url    string `json:"url"`
	Avatar string `json:"avatar_url"`
}

type OrgResource struct {
	client *Client
}

func (r *OrgResource) List() ([]*Org, error) {
	orgs := []*Org{}
	if err := r.client.do("GET", "/user/orgs", nil, &orgs); err != nil {
		return nil, err
	}

	return orgs, nil
}
