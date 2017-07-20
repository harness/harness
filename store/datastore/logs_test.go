package datastore

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/drone/drone/model"
)

func TestLogCreateFind(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from logs")
		s.Close()
	}()

	proc := model.Proc{
		ID: 1,
	}
	buf := bytes.NewBufferString("echo hi")
	err := s.LogSave(&proc, buf)
	if err != nil {
		t.Errorf("Unexpected error: log create: %s", err)
	}

	rc, err := s.LogFind(&proc)
	if err != nil {
		t.Errorf("Unexpected error: log create: %s", err)
	}

	defer rc.Close()
	out, _ := ioutil.ReadAll(rc)
	if got, want := string(out), "echo hi"; got != want {
		t.Errorf("Want log data %s, got %s", want, got)
	}
}

func TestLogUpdate(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from logs")
		s.Close()
	}()

	proc := model.Proc{
		ID: 1,
	}
	buf1 := bytes.NewBufferString("echo hi")
	buf2 := bytes.NewBufferString("echo allo?")
	err1 := s.LogSave(&proc, buf1)
	err2 := s.LogSave(&proc, buf2)
	if err1 != nil {
		t.Errorf("Unexpected error: log create: %s", err1)
	}
	if err2 != nil {
		t.Errorf("Unexpected error: log update: %s", err2)
	}

	rc, err := s.LogFind(&proc)
	if err != nil {
		t.Errorf("Unexpected error: log create: %s", err)
	}

	defer rc.Close()
	out, _ := ioutil.ReadAll(rc)
	if got, want := string(out), "echo allo?"; got != want {
		t.Errorf("Want log data %s, got %s", want, got)
	}
}
