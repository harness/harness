package card

import (
	"bytes"
	"context"
	"database/sql"
	"io/ioutil"
	"testing"

	"github.com/drone/drone/core"
	"github.com/drone/drone/store/build"
	"github.com/drone/drone/store/repos"
	"github.com/drone/drone/store/shared/db/dbtest"
	"github.com/drone/drone/store/step"
)

var noContext = context.TODO()

func TestCard(t *testing.T) {
	conn, err := dbtest.Connect()
	if err != nil {
		t.Error(err)
		return
	}
	defer func() {
		dbtest.Reset(conn)
		dbtest.Disconnect(conn)
	}()

	// seed with a dummy repository
	dummyRepo := &core.Repository{UID: "1", Slug: "octocat/hello-world"}
	repos := repos.New(conn)
	repos.Create(noContext, dummyRepo)

	// seed with a dummy stage
	stage := &core.Stage{Number: 1}
	stages := []*core.Stage{stage}

	// seed with a dummy build
	dummyBuild := &core.Build{Number: 1, RepoID: dummyRepo.ID}
	builds := build.New(conn)
	builds.Create(noContext, dummyBuild, stages)

	// seed with a dummy step
	dummyStep := &core.Step{Number: 1, StageID: stage.ID}
	steps := step.New(conn)
	steps.Create(noContext, dummyStep)

	store := New(conn).(*cardStore)
	t.Run("Create", testCardCreate(store, dummyStep))
	t.Run("Find", testFindCard(store, dummyStep))
	t.Run("Update", testLogsUpdate(store, dummyStep))
}

func testCardCreate(store *cardStore, step *core.Step) func(t *testing.T) {
	return func(t *testing.T) {
		buf := ioutil.NopCloser(
			bytes.NewBuffer([]byte("{\"type\": \"AdaptiveCard\"}")),
		)
		err := store.Create(noContext, step.ID, buf)
		if err != nil {
			t.Error(err)
		}
	}
}

func testFindCard(card *cardStore, step *core.Step) func(t *testing.T) {
	return func(t *testing.T) {
		r, err := card.Find(noContext, step.ID)
		if err != nil {
			t.Error(err)
		} else {
			data, err := ioutil.ReadAll(r)
			if err != nil {
				t.Error(err)
				return
			}
			if got, want := string(data), "{\"type\": \"AdaptiveCard\"}"; got != want {
				t.Errorf("Want card data output stream %q, got %q", want, got)
			}
		}
	}
}

func testLogsUpdate(store *cardStore, step *core.Step) func(t *testing.T) {
	return func(t *testing.T) {
		buf := bytes.NewBufferString("hola mundo")
		err := store.Update(noContext, step.ID, buf)
		if err != nil {
			t.Error(err)
			return
		}
		r, err := store.Find(noContext, step.ID)
		if err != nil {
			t.Error(err)
			return
		}
		data, err := ioutil.ReadAll(r)
		if err != nil {
			t.Error(err)
			return
		}
		if got, want := string(data), "hola mundo"; got != want {
			t.Errorf("Want updated log output stream %q, got %q", want, got)
		}
	}
}

func testLogsDelete(store *cardStore, step *core.Step) func(t *testing.T) {
	return func(t *testing.T) {
		err := store.Delete(noContext, step.ID)
		if err != nil {
			t.Error(err)
			return
		}
		_, err = store.Find(noContext, step.ID)
		if got, want := sql.ErrNoRows, err; got != want {
			t.Errorf("Want sql.ErrNoRows, got %v", got)
			return
		}
	}
}
