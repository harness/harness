package datastore

import (
	"testing"

	"github.com/drone/drone/model"
	"github.com/franela/goblin"
)

func Test_pollstore(t *testing.T) {
	db := openTest()
	defer db.Close()

	s := From(db)
	g := goblin.Goblin(t)
	g.Describe("Polls", func() {
		g.BeforeEach(func() {
			db.Exec(rebind("DELETE FROM `polls`"))
		})

		g.It("Should create poll", func() {
			poll := model.Poll{
				ID:     1,
				Owner:  "user1",
				Name:   "repo1",
				Period: 300,
			}
			err := s.Polls().Create(&poll)
			g.Assert(err == nil).IsTrue()
			g.Assert(poll.ID != 0).IsTrue()
			g.Assert(poll.Owner == "user1").IsTrue()
			g.Assert(poll.Period, 300).IsTrue()
		})

		g.It("Should update a poll", func() {
			poll := modle.Poll{
				ID:     1,
				Period: 600,
			}
			err := s.Polls().Create(&poll)
			g.Assert(err == nil).IsTrue()
			g.Assert(poll.ID != 0).IsTrue()

			poll.Period = 300

			updateErr := s.Polls().Update(&poll)
			updatedPoll, getErr := s.Polls().Get(&model.Repo{ID: 1})
			g.Assert(updateErr == nil).IsTrue()
			g.Assert(getErr == nil).IsTrue()
			g.Assert(poll.ID).Equal(updatedPoll.ID)
			g.Assert(poll.Period).Equal(updatedPoll.ID)
		})

		g.It("Should delete a poll", func() {
			poll := model.Poll{
				ID:     1,
				Period: 300,
			}
			err1 := s.Polls().Create(&poll)
			err2 := s.Polls().Delete(&poll)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()

			_, err := s.Polls().Get(&model.Repo{ID: 1})
			g.Assert(err == nil).IsFalse()
		})
	})
}
