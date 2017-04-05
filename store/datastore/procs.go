package datastore

import (
	"github.com/drone/drone/model"
	"github.com/drone/drone/store/datastore/sql"
	"github.com/russross/meddler"
)

func (db *datastore) ProcLoad(id int64) (*model.Proc, error) {
	stmt := sql.Lookup(db.driver, "procs-find-id")
	proc := new(model.Proc)
	err := meddler.QueryRow(db, proc, stmt, id)
	return proc, err
}

func (db *datastore) ProcFind(build *model.Build, pid int) (*model.Proc, error) {
	stmt := sql.Lookup(db.driver, "procs-find-build-pid")
	proc := new(model.Proc)
	err := meddler.QueryRow(db, proc, stmt, build.ID, pid)
	return proc, err
}

func (db *datastore) ProcChild(build *model.Build, pid int, child string) (*model.Proc, error) {
	stmt := sql.Lookup(db.driver, "procs-find-build-ppid")
	proc := new(model.Proc)
	err := meddler.QueryRow(db, proc, stmt, build.ID, pid, child)
	return proc, err
}

func (db *datastore) ProcList(build *model.Build) ([]*model.Proc, error) {
	stmt := sql.Lookup(db.driver, "procs-find-build")
	list := []*model.Proc{}
	err := meddler.QueryAll(db, &list, stmt, build.ID)
	return list, err
}

func (db *datastore) ProcCreate(procs []*model.Proc) error {
	for _, proc := range procs {
		if err := meddler.Insert(db, "procs", proc); err != nil {
			return err
		}
	}
	return nil
}

func (db *datastore) ProcUpdate(proc *model.Proc) error {
	return meddler.Update(db, "procs", proc)
}

func (db *datastore) ProcClear(build *model.Build) (err error) {
	stmt1 := sql.Lookup(db.driver, "files-delete-build")
	stmt2 := sql.Lookup(db.driver, "procs-delete-build")
	_, err = db.Exec(stmt1, build.ID)
	if err != nil {
		return
	}
	_, err = db.Exec(stmt2, build.ID)
	return
}
