package datastore

import (
	"testing"

	"github.com/drone/drone/model"
)

func TestConfig(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from config")
		s.Close()
	}()

	var (
		data = "pipeline: [ { image: golang, commands: [ go build, go test ] } ]"
		hash = "8d8647c9aa90d893bfb79dddbe901f03e258588121e5202632f8ae5738590b26"
	)

	if err := s.ConfigInsert(
		&model.Config{
			RepoID:   2,
			Data:     data,
			Hash:     hash,
			Approved: false,
		},
	); err != nil {
		t.Errorf("Unexpected error: insert config: %s", err)
		return
	}

	config, err := s.ConfigFind(&model.Repo{ID: 2}, hash)
	if err != nil {
		t.Error(err)
		return
	}
	if got, want := config.ID, int64(1); got != want {
		t.Errorf("Want config id %d, got %d", want, got)
	}
	if got, want := config.RepoID, int64(2); got != want {
		t.Errorf("Want config repo id %d, got %d", want, got)
	}
	if got, want := config.Data, data; got != want {
		t.Errorf("Want config data %s, got %s", want, got)
	}
	if got, want := config.Hash, hash; got != want {
		t.Errorf("Want config hash %s, got %s", want, got)
	}
	if got, want := config.Approved, false; got != want {
		t.Errorf("Want config approved %v, got %v", want, got)
	}

	config.Approved = true
	err = s.ConfigUpdate(config)
	if err != nil {
		t.Errorf("Want config updated, got error %q", err)
		return
	}

	updated, err := s.ConfigFind(&model.Repo{ID: 2}, hash)
	if err != nil {
		t.Errorf("Want config find, got error %q", err)
		return
	}
	if got, want := updated.Approved, true; got != want {
		t.Errorf("Want config approved updated %v, got %v", want, got)
	}
}

//
// func TestConfigIndexes(t *testing.T) {
// 	s := newTest()
// 	defer func() {
// 		s.Exec("delete from config")
// 		s.Close()
// 	}()
//
// 	if err := s.FileCreate(
// 		&model.File{
// 			BuildID: 1,
// 			ProcID:  1,
// 			Name:    "hello.txt",
// 			Size:    11,
// 			Mime:    "text/plain",
// 		},
// 		bytes.NewBufferString("hello world"),
// 	); err != nil {
// 		t.Errorf("Unexpected error: insert file: %s", err)
// 		return
// 	}
//
// 	// fail due to duplicate file name
// 	if err := s.FileCreate(
// 		&model.File{
// 			BuildID: 1,
// 			ProcID:  1,
// 			Name:    "hello.txt",
// 			Mime:    "text/plain",
// 			Size:    11,
// 		},
// 		bytes.NewBufferString("hello world"),
// 	); err == nil {
// 		t.Errorf("Unexpected error: dupliate pid")
// 	}
// }
