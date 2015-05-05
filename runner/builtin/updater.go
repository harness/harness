package builtin

import (
	"io"
	"io/ioutil"

	"github.com/drone/drone/common"
	"github.com/drone/drone/datastore"
	"github.com/drone/drone/eventbus"
)

type Updater interface {
	SetBuild(*common.Repo, *common.Build) error
	SetTask(*common.Repo, *common.Build, *common.Task) error
	SetLogs(*common.Repo, *common.Build, *common.Task, io.ReadCloser) error
}

// NewUpdater returns an implementation of the Updater interface
// that directly modifies the database and sends messages to the bus.
func NewUpdater(bus eventbus.Bus, store datastore.Datastore) Updater {
	return &updater{bus, store}
}

type updater struct {
	bus   eventbus.Bus
	store datastore.Datastore
}

func (u *updater) SetBuild(r *common.Repo, b *common.Build) error {
	err := u.store.SetBuildState(r.FullName, b)
	if err != nil {
		return err
	}
	u.bus.Send(&eventbus.Event{
		Repo:  r,
		Build: b,
	})
	return nil
}

func (u *updater) SetTask(r *common.Repo, b *common.Build, t *common.Task) error {
	err := u.store.SetBuildTask(r.FullName, b.Number, t)
	if err != nil {
		return err
	}
	u.bus.Send(&eventbus.Event{
		Repo:  r,
		Build: b,
		Task:  t,
	})
	return nil
}

func (u *updater) SetLogs(r *common.Repo, b *common.Build, t *common.Task, rc io.ReadCloser) error {
	defer rc.Close()
	out, err := ioutil.ReadAll(rc)
	if err != nil {
		return err
	}
	return u.store.SetLogs(r.FullName, b.Number, t.Number, out)
}
