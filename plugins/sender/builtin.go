package sender

import (
	"github.com/drone/drone/model"
)

type builtin struct {
	store model.SenderStore
}

// New returns a new local gating service.
func New(store model.SenderStore) model.SenderService {
	return &builtin{store}
}

func (b *builtin) SenderAllowed(user *model.User, repo *model.Repo, build *model.Build) (bool, error) {
	if repo.IsPrivate == false && build.Event == model.EventPull && build.Sender != user.Login {
		sender, err := b.store.SenderFind(repo, build.Sender)
		if err != nil || sender.Block {
			return false, nil
		}
	}
	return true, nil
}

func (b *builtin) SenderCreate(repo *model.Repo, sender *model.Sender) error {
	return b.store.SenderCreate(sender)
}

func (b *builtin) SenderUpdate(repo *model.Repo, sender *model.Sender) error {
	return b.store.SenderUpdate(sender)
}

func (b *builtin) SenderDelete(repo *model.Repo, login string) error {
	sender, err := b.store.SenderFind(repo, login)
	if err != nil {
		return err
	}
	return b.store.SenderDelete(sender)
}

func (b *builtin) SenderList(repo *model.Repo) ([]*model.Sender, error) {
	return b.store.SenderList(repo)
}
