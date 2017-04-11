package datastore

import (
	"testing"

	"github.com/drone/drone/model"
)

func TestSenderFind(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from senders")
		s.Close()
	}()

	err := s.SenderCreate(&model.Sender{
		RepoID: 1,
		Login:  "octocat",
		Allow:  true,
		Block:  false,
	})
	if err != nil {
		t.Errorf("Unexpected error: insert secret: %s", err)
		return
	}

	sender, err := s.SenderFind(&model.Repo{ID: 1}, "octocat")
	if err != nil {
		t.Error(err)
		return
	}
	if got, want := sender.RepoID, int64(1); got != want {
		t.Errorf("Want repo id %d, got %d", want, got)
	}
	if got, want := sender.Login, "octocat"; got != want {
		t.Errorf("Want sender login %s, got %s", want, got)
	}
	if got, want := sender.Allow, true; got != want {
		t.Errorf("Want sender allow %v, got %v", want, got)
	}
}

func TestSenderList(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from senders")
		s.Close()
	}()

	s.SenderCreate(&model.Sender{
		RepoID: 1,
		Login:  "octocat",
		Allow:  true,
		Block:  false,
	})
	s.SenderCreate(&model.Sender{
		RepoID: 1,
		Login:  "defunkt",
		Allow:  true,
		Block:  false,
	})

	list, err := s.SenderList(&model.Repo{ID: 1})
	if err != nil {
		t.Error(err)
		return
	}
	if got, want := len(list), 2; got != want {
		t.Errorf("Want %d senders, got %d", want, got)
	}
}

func TestSenderUpdate(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from senders")
		s.Close()
	}()

	sender := &model.Sender{
		RepoID: 1,
		Login:  "octocat",
		Allow:  true,
		Block:  false,
	}
	if err := s.SenderCreate(sender); err != nil {
		t.Errorf("Unexpected error: insert sender: %s", err)
		return
	}
	sender.Allow = false
	if err := s.SenderUpdate(sender); err != nil {
		t.Errorf("Unexpected error: update sender: %s", err)
		return
	}
	updated, err := s.SenderFind(&model.Repo{ID: 1}, "octocat")
	if err != nil {
		t.Error(err)
		return
	}
	if got, want := updated.Allow, false; got != want {
		t.Errorf("Want allow value %v, got %v", want, got)
	}
}

func TestSenderIndexes(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from senders")
		s.Close()
	}()

	if err := s.SenderCreate(&model.Sender{
		RepoID: 1,
		Login:  "octocat",
		Allow:  true,
		Block:  false,
	}); err != nil {
		t.Errorf("Unexpected error: insert sender: %s", err)
		return
	}

	// fail due to duplicate name
	if err := s.SenderCreate(&model.Sender{
		RepoID: 1,
		Login:  "octocat",
		Allow:  true,
		Block:  false,
	}); err == nil {
		t.Errorf("Unexpected error: dupliate login")
	}
}
