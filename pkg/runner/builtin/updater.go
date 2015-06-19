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
	SetCommit(*types.User, *types.Repo, *types.Commit) error
	SetJob(*types.Repo, *types.Commit, *types.Job) error
	SetLogs(*types.Repo, *types.Commit, *types.Job, io.ReadCloser) error
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

func (u *updater) SetCommit(user *types.User, r *types.Repo, c *types.Commit) error {
	err := u.store.SetCommit(c)
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
	if c.Builds == nil || len(c.Builds) == 0 {
		c.Builds, _ = u.store.JobList(c)
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

func (u *updater) SetJob(r *types.Repo, c *types.Commit, j *types.Job) error {
	err := u.store.SetJob(j)
	if err != nil {
		return err
	}

	// we need this because builds coming from
	// a remote agent won't have the embedded
	// build list. we should probably just rethink
	// the messaging instead of this hack.
	if c.Builds == nil || len(c.Builds) == 0 {
		c.Builds, _ = u.store.JobList(c)
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

func (u *updater) SetLogs(r *types.Repo, c *types.Commit, j *types.Job, rc io.ReadCloser) error {
	path := fmt.Sprintf("/logs/%s/%v/%v", r.FullName, c.Sequence, j.Number)
	return u.store.SetBlobReader(path, rc)
}
