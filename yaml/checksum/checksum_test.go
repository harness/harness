package checksum

import (
	"testing"

	"github.com/franela/goblin"
)

func TestParse(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Shasum", func() {

		g.It("Should parse the shasum string", func() {
			hash, _, _ := split("f1d2d2f924e986ac86fdf7b36c94bcdf32beec15")
			g.Assert(hash).Equal("f1d2d2f924e986ac86fdf7b36c94bcdf32beec15")
		})

		g.It("Should parse a two-part shasum string", func() {
			hash, _, name := split("f1d2d2f924e986ac86fdf7b36c94bcdf32beec15 .drone.yml")
			g.Assert(hash).Equal("f1d2d2f924e986ac86fdf7b36c94bcdf32beec15")
			g.Assert(name).Equal(".drone.yml")
		})

		g.It("Should parse a three-part shasum string", func() {
			hash, size, name := split("f1d2d2f924e986ac86fdf7b36c94bcdf32beec15 42 .drone.yml")
			g.Assert(hash).Equal("f1d2d2f924e986ac86fdf7b36c94bcdf32beec15")
			g.Assert(name).Equal(".drone.yml")
			g.Assert(size).Equal(int64(42))
		})

		g.It("Should calc a sha1 sum", func() {
			hash := sha1sum("foo\n")
			g.Assert(hash).Equal("f1d2d2f924e986ac86fdf7b36c94bcdf32beec15")
		})

		g.It("Should calc a sha256 sum", func() {
			hash := sha256sum("foo\n")
			g.Assert(hash).Equal("b5bb9d8014a0f9b1d61e21e796d78dccdf1352f23cd32812f4850b878ae4944c")
		})

		g.It("Should calc a sha512 sum", func() {
			hash := sha512sum("foo\n")
			g.Assert(hash).Equal("0cf9180a764aba863a67b6d72f0918bc131c6772642cb2dce5a34f0a702f9470ddc2bf125c12198b1995c233c34b4afd346c54a2334c350a948a51b6e8b4e6b6")
		})

		g.It("Should calc a sha1 sum", func() {
			hash := sha1sum("foo\n")
			g.Assert(hash).Equal("f1d2d2f924e986ac86fdf7b36c94bcdf32beec15")
		})

		g.It("Should validate sha1 sum with file size", func() {
			ok := Check("foo\n", "f1d2d2f924e986ac86fdf7b36c94bcdf32beec15 4 -")
			g.Assert(ok).IsTrue()
		})

		g.It("Should validate sha256 sum with file size", func() {
			ok := Check("foo\n", "b5bb9d8014a0f9b1d61e21e796d78dccdf1352f23cd32812f4850b878ae4944c 4 -")
			g.Assert(ok).IsTrue()
		})

		g.It("Should validate sha512 sum with file size", func() {
			ok := Check("foo\n", "0cf9180a764aba863a67b6d72f0918bc131c6772642cb2dce5a34f0a702f9470ddc2bf125c12198b1995c233c34b4afd346c54a2334c350a948a51b6e8b4e6b6 4 -")
			g.Assert(ok).IsTrue()
		})

		g.It("Should fail validation if incorrect sha1", func() {
			ok := Check("bar\n", "f1d2d2f924e986ac86fdf7b36c94bcdf32beec15 4 -")
			g.Assert(ok).IsFalse()
		})

		g.It("Should fail validation if incorrect sha256", func() {
			ok := Check("bar\n", "b5bb9d8014a0f9b1d61e21e796d78dccdf1352f23cd32812f4850b878ae4944c 4 -")
			g.Assert(ok).IsFalse()
		})

		g.It("Should fail validation if incorrect sha512", func() {
			ok := Check("bar\n", "0cf9180a764aba863a67b6d72f0918bc131c6772642cb2dce5a34f0a702f9470ddc2bf125c12198b1995c233c34b4afd346c54a2334c350a948a51b6e8b4e6b6 4 -")
			g.Assert(ok).IsFalse()
		})

		g.It("Should return false if file size mismatch", func() {
			ok := Check("foo\n", "f1d2d2f924e986ac86fdf7b36c94bcdf32beec15 12 -")
			g.Assert(ok).IsFalse()
		})

		g.It("Should return false if invalid checksum string", func() {
			ok := Check("foo\n", "f1d2d2f924e986ac86fdf7b36c94bcdf32beec15234")
			g.Assert(ok).IsFalse()
		})

		g.It("Should return false if empty checksum", func() {
			ok := Check("foo\n", "")
			g.Assert(ok).IsFalse()
		})
	})
}
