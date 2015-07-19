package builtin

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/drone/drone/pkg/bus"
	"github.com/drone/drone/pkg/remote"
	"github.com/drone/drone/pkg/store"
	"github.com/drone/drone/pkg/types"
)

type Updater interface {
	SetBuild(*types.User, *types.Repo, *types.Build) error
	SetJob(*types.Repo, *types.Build, *types.Job) error
	SetLogs(*types.Repo, *types.Build, *types.Job, io.ReadCloser) error
	GetBuild(int64) *types.Build
}

// NewUpdater returns an implementation of the Updater interface
// that directly modifies the database and sends messages to the bus.
func NewUpdater(bus bus.Bus, store store.Store, rem remote.Remote) Updater {
	return &updater{bus, store, rem}
}

type updater struct {
	bus    bus.Bus
	store  store.Store
	remote remote.Remote
}

func (u *updater) GetBuild(id int64) *types.Build {
	return u.store.Build(id)
}

func (u *updater) SetBuild(user *types.User, r *types.Repo, c *types.Build) error {
	err := u.store.SetBuild(c)
	if err != nil {
		return err
	}

	err = u.remote.Status(user, r, c)
	if err != nil {
		// log err
	}

	// we need this because builds coming from
	// a remote agent won't have the embedded
	// build list. we should probably just rethink
	// the messaging instead of this hack.
	if c.Jobs == nil || len(c.Jobs) == 0 {
		c.Jobs, _ = u.store.JobList(c)
	}

	msg, err := json.Marshal(c)
	if err != nil {
		return err
	}

	u.bus.Send(&bus.Event{
		Name: r.FullName,
		Kind: bus.EventRepo,
		Msg:  msg,
	})
	return nil
}

func (u *updater) SetJob(r *types.Repo, c *types.Build, j *types.Job) error {
	err := u.store.SetJob(j)
	if err != nil {
		return err
	}

	// we need this because builds coming from
	// a remote agent won't have the embedded
	// build list. we should probably just rethink
	// the messaging instead of this hack.
	if c.Jobs == nil || len(c.Jobs) == 0 {
		c.Jobs, _ = u.store.JobList(c)
	}

	msg, err := json.Marshal(c)
	if err != nil {
		return err
	}

	u.bus.Send(&bus.Event{
		Name: r.FullName,
		Kind: bus.EventRepo,
		Msg:  msg,
	})
	return nil
}

func (u *updater) SetLogs(r *types.Repo, c *types.Build, j *types.Job, rc io.ReadCloser) error {
	path := fmt.Sprintf("/logs/%s/%v/%v", r.FullName, c.Number, j.Number)
	return u.store.SetBlobReader(path, rc)
}
