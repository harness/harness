package database

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/franela/goblin"
)

func TestBlobstore(t *testing.T) {
	db := mustConnectTest()
	bs := NewBlobstore(db)
	defer db.Close()

	g := goblin.Goblin(t)
	g.Describe("Blobstore", func() {

		// before each test be sure to purge the package
		// table data from the database.
		g.BeforeEach(func() {
			db.Exec("DELETE FROM blobs")
		})

		g.It("Should Put a Blob", func() {
			err := bs.Put("foo", []byte("bar"))
			g.Assert(err == nil).IsTrue()
		})

		g.It("Should Put a Blob reader", func() {
			var buf bytes.Buffer
			buf.Write([]byte("bar"))
			err := bs.PutReader("foo", &buf)
			g.Assert(err == nil).IsTrue()
		})

		g.It("Should Overwrite a Blob", func() {
			bs.Put("foo", []byte("bar"))
			bs.Put("foo", []byte("baz"))
			blob, err := bs.Get("foo")
			g.Assert(err == nil).IsTrue()
			g.Assert(string(blob)).Equal("baz")
		})

		g.It("Should Get a Blob", func() {
			bs.Put("foo", []byte("bar"))
			blob, err := bs.Get("foo")
			g.Assert(err == nil).IsTrue()
			g.Assert(string(blob)).Equal("bar")
		})

		g.It("Should Get a Blob reader", func() {
			bs.Put("foo", []byte("bar"))
			r, _ := bs.GetReader("foo")
			blob, _ := ioutil.ReadAll(r)
			g.Assert(string(blob)).Equal("bar")
		})

		g.It("Should Del a Blob", func() {
			bs.Put("foo", []byte("bar"))
			err := bs.Del("foo")
			g.Assert(err == nil).IsTrue()
		})

	})
}
