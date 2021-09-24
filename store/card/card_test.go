package card

import (
	"context"
	"database/sql"
	"io/ioutil"
	"testing"

	"github.com/drone/drone/core"
	"github.com/drone/drone/store/shared/db/dbtest"
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

	store := New(conn).(*cardStore)
	t.Run("Create", testCardCreate(store))
}

func testCard(item *core.Card) func(t *testing.T) {
	return func(t *testing.T) {
		if got, want := item.Schema, "https://myschema.com"; got != want {
			t.Errorf("Want card schema %q, got %q", want, got)
		}
		if got, want := item.Build, int64(1); got != want {
			t.Errorf("Want card build number %q, got %q", want, got)
		}
		if got, want := item.Stage, int64(2); got != want {
			t.Errorf("Want card stage number %q, got %q", want, got)
		}
		if got, want := item.Step, int64(3); got != want {
			t.Errorf("Want card step number %q, got %q", want, got)
		}
	}
}

func testCardCreate(store *cardStore) func(t *testing.T) {
	return func(t *testing.T) {
		item := &core.CreateCard{
			Id:     1,
			Build:  1,
			Stage:  2,
			Step:   3,
			Schema: "https://myschema.com",
			Data:   "{\"type\": \"AdaptiveCard\"}",
		}
		err := store.CreateCard(noContext, item)
		if err != nil {
			t.Error(err)
		}
		if item.Id == 0 {
			t.Errorf("Want card Id assigned, got %d", item.Id)
		}

		t.Run("FindByBuild", testFindCardByBuild(store))
		t.Run("FindCard", testFindCard(store))
		t.Run("FindCardData", testFindCardData(store))
		t.Run("Delete", testCardDelete(store))
	}
}

func testFindCardByBuild(card *cardStore) func(t *testing.T) {
	return func(t *testing.T) {
		item, err := card.FindCardByBuild(noContext, 1)
		if err != nil {
			t.Error(err)
		} else {
			t.Run("Fields", testCard(item[0]))
		}
	}
}

func testFindCard(card *cardStore) func(t *testing.T) {
	return func(t *testing.T) {
		item, err := card.FindCard(noContext, 3)
		if err != nil {
			t.Error(err)
		} else {
			t.Run("Fields", testCard(item))
		}
	}
}

func testFindCardData(card *cardStore) func(t *testing.T) {
	return func(t *testing.T) {
		r, err := card.FindCardData(noContext, 1)
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

func testCardDelete(store *cardStore) func(t *testing.T) {
	return func(t *testing.T) {
		card, err := store.FindCard(noContext, 3)
		if err != nil {
			t.Error(err)
			return
		}
		err = store.DeleteCard(noContext, card.Id)
		if err != nil {
			t.Error(err)
			return
		}
		_, err = store.FindCard(noContext, card.Step)
		if got, want := sql.ErrNoRows, err; got != want {
			t.Errorf("Want sql.ErrNoRows, got %v", got)
			return
		}
	}
}
