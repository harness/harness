// Copyright 2018 Drone.IO Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package datastore

import (
	"testing"

	"github.com/drone/drone/model"
)

func TestTaskList(t *testing.T) {
	s := newTest()
	defer func() {
		s.Exec("delete from tasks")
		s.Close()
	}()

	s.TaskInsert(&model.Task{
		ID:     "some_random_id",
		Data:   []byte("foo"),
		Labels: map[string]string{"foo": "bar"},
	})

	list, err := s.TaskList()
	if err != nil {
		t.Error(err)
		return
	}
	if got, want := len(list), 1; got != want {
		t.Errorf("Want %d task, got %d", want, got)
		return
	}
	if got, want := list[0].ID, "some_random_id"; got != want {
		t.Errorf("Want task id %s, got %s", want, got)
	}
	if got, want := list[0].Data, "foo"; string(got) != want {
		t.Errorf("Want task data %s, got %s", want, string(got))
	}

	err = s.TaskDelete("some_random_id")
	if err != nil {
		t.Error(err)
		return
	}

	list, err = s.TaskList()
	if err != nil {
		t.Error(err)
		return
	}
	if got, want := len(list), 0; got != want {
		t.Errorf("Want empty task list after delete")
	}
}
