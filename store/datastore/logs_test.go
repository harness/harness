package datastore

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/drone/drone/model"
	"github.com/franela/goblin"
)

func TestLogs(t *testing.T) {
	db := openTest()
	defer db.Close()

	s := From(db)
	g := goblin.Goblin(t)
	g.Describe("Logs", func() {

		// before each test be sure to purge the package
		// table data from the database.
		g.BeforeEach(func() {
			db.Exec("DELETE FROM logs")
		})

		g.It("Should create a log", func() {
			job := model.Job{
				ID: 1,
			}
			buf := bytes.NewBufferString("echo hi")
			err := s.WriteLog(&job, buf)
			g.Assert(err == nil).IsTrue()

			rc, err := s.ReadLog(&job)
			g.Assert(err == nil).IsTrue()
			defer rc.Close()
			out, _ := ioutil.ReadAll(rc)
			g.Assert(string(out)).Equal("echo hi")
		})

		g.It("Should update a log", func() {
			job := model.Job{
				ID: 1,
			}
			buf1 := bytes.NewBufferString("echo hi")
			buf2 := bytes.NewBufferString("echo allo?")
			err1 := s.WriteLog(&job, buf1)
			err2 := s.WriteLog(&job, buf2)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()

			rc, err := s.ReadLog(&job)
			g.Assert(err == nil).IsTrue()
			defer rc.Close()
			out, _ := ioutil.ReadAll(rc)
			g.Assert(string(out)).Equal("echo allo?")
		})

	})
}
