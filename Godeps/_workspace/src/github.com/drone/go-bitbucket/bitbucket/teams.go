package bitbucket

const (
	TeamRoleAdmin = "admin"
	TeamRoleCollab = "collaborator"
)

// Use this resource to manage privilege settings for a team account. Team
// accounts can grant groups account privileges as well as repository access.
// Groups with account privileges are those with can administer this account
// (admin rights) or can create repositories in this account (collaborator
// rights) checked.
//
// https://confluence.atlassian.com/display/BITBUCKET/privileges+Resource
type TeamResource struct {
	client *Client
}

type Team struct {
	// The team or individual account name.
	Name string

	// The group's slug.
	Role string

}

// Gets the groups with account privileges defined for a team account.
func (r *TeamResource) List() ([]*Team, error) {
	
	// we'll get the data in a key/value struct
	data := struct {
		Teams map[string]string
	}{ }

	data.Teams = map[string]string{}
	teams := []*Team{}

	if err := r.client.do("GET", "/user/privileges", nil, nil, &data); err != nil {
		return nil, err
	}

	for k,v := range data.Teams {
		team := &Team{ k, v }
		teams = append(teams, team)
	}

	return teams, nil
}
