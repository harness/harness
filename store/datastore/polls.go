package datastore

import (
	"database/sql"

	"github.com/drone/drone/model"
	"github.com/russross/meddler"
)

type pollstore struct {
	*sql.DB
}

func (db *pollstore) Get(poll *model.Poll) (*model.Poll, error) {
	var p = new(model.Poll)
	var err = meddler.QueryRow(db, p, rebind(pollQuery), poll.Owner, poll.Name)
	return poll, err
}

func (db *pollstore) Create(poll *model.Poll) error {
	return meddler.Save(db, pollTable, poll)
}

func (db *pollstore) Update(poll *model.Poll) error {
	return meddler.Save(db, pollTable, poll)
}

func (db *pollstore) Delete(poll *model.Poll) error {
	var _, err = db.Exec(rebind(pollDeleteStmt), poll.ID)
	return err
}

func (db *pollstore) GetPollList() ([]*model.Poll, error) {
	var polls []*model.Poll
	err := meddler.QueryAll(db, &polls, pollQueryAll, []interface{}{}...)
	return polls, err
}

const pollTable = "polls"

const pollQuery = "SELECT * FROM polls WHERE owner=? and name=? LIMIT 1"

const pollDeleteStmt = "DELETE FROM polls WHERE id=?"

const pollQueryAll = "SELECT * FROM polls"
