package sender

import (
	"fmt"

	"github.com/drone/drone/model"
	"github.com/drone/drone/plugins/internal"
)

type plugin struct {
	endpoint string
}

// NewRemote returns a new remote gating service.
func NewRemote(endpoint string) model.SenderService {
	return &plugin{endpoint}
}

func (p *plugin) SenderAllowed(user *model.User, repo *model.Repo, build *model.Build) (bool, error) {
	path := fmt.Sprintf("%s/sender/%s/%s/%s", p.endpoint, repo.Owner, repo.Name, build.Sender)
	out := new(model.Sender)
	err := internal.Send("POST", path, build, out)
	if err != nil {
		return false, err
	}
	return out.Allow, nil
}

func (p *plugin) SenderCreate(repo *model.Repo, sender *model.Sender) error {
	path := fmt.Sprintf("%s/sender/%s/%s", p.endpoint, repo.Owner, repo.Name)
	return internal.Send("POST", path, sender, nil)
}

func (p *plugin) SenderUpdate(repo *model.Repo, sender *model.Sender) error {
	path := fmt.Sprintf("%s/sender/%s/%s", p.endpoint, repo.Owner, repo.Name)
	return internal.Send("PUT", path, sender, nil)
}

func (p *plugin) SenderDelete(repo *model.Repo, login string) error {
	path := fmt.Sprintf("%s/sender/%s/%s/%s", p.endpoint, repo.Owner, repo.Name, login)
	return internal.Send("DELETE", path, nil, nil)
}

func (p *plugin) SenderList(repo *model.Repo) ([]*model.Sender, error) {
	path := fmt.Sprintf("%s/sender/%s/%s", p.endpoint, repo.Owner, repo.Name)
	out := []*model.Sender{}
	err := internal.Send("GET", path, nil, out)
	return out, err
}
