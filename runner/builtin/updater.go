package builtin

import (
	"encoding/json"
	"io"

	"github.com/drone/drone/common"
	"github.com/drone/drone/datastore"
	"github.com/drone/drone/eventbus"
	"github.com/drone/drone/remote"
)

type Updater interface {
	SetBuild(*common.User, *common.Repo, *common.Build) error
	SetTask(*common.Repo, *common.Build, *common.Task) error
	SetLogs(*common.Repo, *common.Build, *common.Task, io.ReadCloser) error
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

func (u *updater) SetBuild(user *common.User, r *common.Repo, b *common.Build) error {
	err := u.store.SetBuildState(r.FullName, b)
	if err != nil {
		return err
	}

	// if the build is complete we may need to update
	if b.State != common.StatePending && b.State != common.StateRunning {
		repo, err := u.store.Repo(r.FullName)
		if err == nil {
			if repo.Last == nil || b.Number >= repo.Last.Number {
				repo.Last = b
				u.store.SetRepo(repo)
			}
		}

		// err = u.remote.Status(user, r, b, "")
		// if err != nil {
		//
		// }
	}

	msg, err := json.Marshal(b)
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

func (u *updater) SetTask(r *common.Repo, b *common.Build, t *common.Task) error {
	err := u.store.SetBuildTask(r.FullName, b.Number, t)
	if err != nil {
		return err
	}

	msg, err := json.Marshal(b)
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

func (u *updater) SetLogs(r *common.Repo, b *common.Build, t *common.Task, rc io.ReadCloser) error {
	//defer rc.Close()
	//out, err := ioutil.ReadAll(rc)
	//if err != nil {
	//	return err
	//}
	//return u.store.SetLogs(r.FullName, b.Number, t.Number, out)
	return u.store.SetLogs(r.FullName, b.Number, t.Number, rc)
}
