package secure

import (
	"testing"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/franela/goblin"

	"github.com/drone/drone/pkg/utils/sshutil"
)

func Test_Secure(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Encrypt params", func() {
		privKey, _ := sshutil.GeneratePrivateKey()
		publicKey := &privKey.PublicKey

		privateKeyPEM := string(sshutil.MarshalPrivateKey(privKey))

		repoHash := "9T2tH3qZ8FSPr9uxrhzV4mn2VdVgA56xPVtYvCh0"
		hashKey := ToHash(repoHash)
		text := "super_duper_secret"
		encryptedValue, _ := sshutil.Encrypt(hashKey, publicKey, text)

		g.It("Should decrypt a yaml", func() {
			yaml := "secure: {\"foo\": \"" + encryptedValue + "\"}"
			decrypted, err := Parse(privateKeyPEM, repoHash, yaml)

			g.Assert(err == nil).IsTrue()
			g.Assert(decrypted["foo"]).Equal(text)
		})

		g.It("Should decrypt a yaml with no secure section", func() {
			yaml := `foo: bar`
			decrypted, err := Parse(privateKeyPEM, repoHash, yaml)
			g.Assert(err == nil).IsTrue()
			g.Assert(len(decrypted)).Equal(0)
		})

		g.It("Should encrypt a map", func() {
			params := map[string]string{
				"foo": text,
			}
			err := EncryptMap(hashKey, publicKey, params)
			g.Assert(err == nil).IsTrue()
			g.Assert(params["foo"] == "super_duper_secret").IsFalse()
			err = DecryptMap(hashKey, privKey, params)
			g.Assert(err == nil).IsTrue()
			g.Assert(params["foo"] == "super_duper_secret").IsTrue()
		})
	})
}
