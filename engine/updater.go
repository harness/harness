package engine

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote"
	"github.com/drone/drone/store"
	"golang.org/x/net/context"
)

type updater struct {
	bus *eventbus
}

func (u *updater) SetBuild(c context.Context, r *Task) error {
	err := store.UpdateBuild(c, r.Build)
	if err != nil {
		return err
	}

	err = remote.FromContext(c).Status(r.User, r.Repo, r.Build, fmt.Sprintf("%s/%s/%d", r.System.Link, r.Repo.FullName, r.Build.Number))
	if err != nil {
		// log err
	}

	msg, err := json.Marshal(&payload{r.Build, r.Jobs})
	if err != nil {
		return err
	}

	u.bus.send(&Event{
		Name: r.Repo.FullName,
		Msg:  msg,
	})
	return nil
}

func (u *updater) SetJob(c context.Context, r *Task) error {
	err := store.UpdateJob(c, r.Job)
	if err != nil {
		return err
	}

	msg, err := json.Marshal(&payload{r.Build, r.Jobs})
	if err != nil {
		return err
	}

	u.bus.send(&Event{
		Name: r.Repo.FullName,
		Msg:  msg,
	})
	return nil
}

func (u *updater) SetLogs(c context.Context, r *Task, rc io.ReadCloser) error {
	return store.WriteLog(c, r.Job, rc)
}

type payload struct {
	*model.Build
	Jobs []*model.Job `json:"jobs"`
}
