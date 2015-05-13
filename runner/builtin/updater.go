package builtin

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/drone/drone/common"
	"github.com/drone/drone/datastore"
	"github.com/drone/drone/eventbus"
	"github.com/drone/drone/remote"
)

type Updater interface {
	SetCommit(*common.User, *common.Repo, *common.Commit) error
	SetBuild(*common.Repo, *common.Commit, *common.Build) error
	SetLogs(*common.Repo, *common.Commit, *common.Build, io.ReadCloser) error
}

// NewUpdater returns an implementation of the Updater interface
// that directly modifies the database and sends messages to the bus.
func NewUpdater(bus eventbus.Bus, store datastore.Datastore, rem remote.Remote) Updater {
	return &updater{bus, store, rem}
}

type updater struct {
	bus    eventbus.Bus
	store  datastore.Datastore
	remote remote.Remote
}

func (u *updater) SetCommit(user *common.User, r *common.Repo, c *common.Commit) error {
	err := u.store.SetCommit(c)
	if err != nil {
		return err
	}

	err = u.remote.Status(user, r, c)
	if err != nil {
		// log err
	}

	msg, err := json.Marshal(c)
	if err != nil {
		return err
	}

	u.bus.Send(&eventbus.Event{
		Name: r.FullName,
		Kind: eventbus.EventRepo,
		Msg:  msg,
	})
	return nil
}

func (u *updater) SetBuild(r *common.Repo, c *common.Commit, b *common.Build) error {
	err := u.store.SetBuild(b)
	if err != nil {
		return err
	}

	msg, err := json.Marshal(c)
	if err != nil {
		return err
	}

	u.bus.Send(&eventbus.Event{
		Name: r.FullName,
		Kind: eventbus.EventRepo,
		Msg:  msg,
	})
	return nil
}

func (u *updater) SetLogs(r *common.Repo, c *common.Commit, b *common.Build, rc io.ReadCloser) error {
	path := fmt.Sprintf("/logs/%s/%v/%v", r.FullName, c.Sequence, b.Sequence)
	return u.store.SetBlobReader(path, rc)
}
