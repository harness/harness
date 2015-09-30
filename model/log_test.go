package model

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/drone/drone/shared/database"
	"github.com/franela/goblin"
)

func TestLog(t *testing.T) {
	db := database.Open("sqlite3", ":memory:")
	defer db.Close()

	g := goblin.Goblin(t)
	g.Describe("Logs", func() {

		// before each test be sure to purge the package
		// table data from the database.
		g.BeforeEach(func() {
			db.Exec("DELETE FROM logs")
		})

		g.It("Should create a log", func() {
			job := Job{
				ID: 1,
			}
			buf := bytes.NewBufferString("echo hi")
			err := SetLog(db, &job, buf)
			g.Assert(err == nil).IsTrue()

			rc, err := GetLog(db, &job)
			g.Assert(err == nil).IsTrue()
			defer rc.Close()
			out, _ := ioutil.ReadAll(rc)
			g.Assert(string(out)).Equal("echo hi")
		})

		g.It("Should update a log", func() {
			job := Job{
				ID: 1,
			}
			buf1 := bytes.NewBufferString("echo hi")
			buf2 := bytes.NewBufferString("echo allo?")
			err1 := SetLog(db, &job, buf1)
			err2 := SetLog(db, &job, buf2)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()

			rc, err := GetLog(db, &job)
			g.Assert(err == nil).IsTrue()
			defer rc.Close()
			out, _ := ioutil.ReadAll(rc)
			g.Assert(string(out)).Equal("echo allo?")
		})

	})
}
