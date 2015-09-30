package engine

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"

	"github.com/CiscoCloud/drone/model"
	"github.com/CiscoCloud/drone/remote"
)

type updater struct {
	bus    *eventbus
	db     *sql.DB
	remote remote.Remote
}

func (u *updater) SetBuild(r *Task) error {
	err := model.UpdateBuild(u.db, r.Build)
	if err != nil {
		return err
	}

	err = u.remote.Status(r.User, r.Repo, r.Build, fmt.Sprintf("%s/%s/%d", r.System.Link, r.Repo.FullName, r.Build.Number))
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

func (u *updater) SetJob(r *Task) error {
	err := model.UpdateJob(u.db, r.Job)
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

func (u *updater) SetLogs(r *Task, rc io.ReadCloser) error {
	return model.SetLog(u.db, r.Job, rc)
}

type payload struct {
	*model.Build
	Jobs []*model.Job `json:"jobs"`
}
