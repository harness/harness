package sshutil

import (
	"crypto/sha256"
	"testing"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/franela/goblin"
)

func TestSSHUtil(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("sshutil", func() {
		var encrypted, testMsg string

		privkey, err := GeneratePrivateKey()
		g.Assert(err == nil).IsTrue()
		pubkey := privkey.PublicKey
		sha256 := sha256.New()
		testMsg = "foo=bar"

		g.Before(func() {
			encrypted, err = Encrypt(sha256, &pubkey, testMsg)
			g.Assert(err == nil).IsTrue()
		})

		g.It("Can decrypt encrypted msg", func() {
			decrypted, err := Decrypt(sha256, privkey, encrypted)
			g.Assert(err == nil).IsTrue()
			g.Assert(decrypted == testMsg).IsTrue()
		})

		g.It("Unmarshals private key from PEM block", func() {
			privateKeyPEM := MarshalPrivateKey(privkey)
			privateKey := UnMarshalPrivateKey(privateKeyPEM)

			g.Assert(privateKey.PublicKey.E == pubkey.E).IsTrue()
		})
	})
}
