package secure

import (
	"testing"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/franela/goblin"

	common "github.com/drone/drone/pkg/types"
	"github.com/drone/drone/pkg/utils/sshutil"
)

func Test_Secure(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Encrypt params", func() {
		privKey, _ := sshutil.GeneratePrivateKey()
		keypair := common.Keypair{
			Private: string(sshutil.MarshalPrivateKey(privKey)),
			Public:  string(sshutil.MarshalPublicKey(&privKey.PublicKey)),
		}
		repo := common.Repo{
			Hash: "9T2tH3qZ8FSPr9uxrhzV4mn2VdVgA56xPVtYvCh0",
			Keys: &keypair,
		}
		hashKey := toHash(repo.Hash)
		text := "super_duper_secret"
		encryptedValue, _ := sshutil.Encrypt(hashKey, &privKey.PublicKey, text)

		g.It("Should decrypt a yaml", func() {
			yaml := "secure: {\"foo\": \"" + encryptedValue + "\"}"
			decrypted, err := Parse(&repo, yaml)

			g.Assert(err == nil).IsTrue()
			g.Assert(decrypted["foo"]).Equal(text)
		})

		g.It("Should decrypt a yaml with no secure section", func() {
			yaml := `foo: bar`
			decrypted, err := Parse(&repo, yaml)
			g.Assert(err == nil).IsTrue()
			g.Assert(len(decrypted)).Equal(0)
		})

		g.It("Should encrypt a map", func() {
			params := map[string]string{
				"foo": text,
			}
			err := EncryptMap(&repo, params)
			g.Assert(err == nil).IsTrue()
			g.Assert(params["foo"] == "super_duper_secret").IsFalse()
			err = DecryptMap(&repo, params)
			g.Assert(err == nil).IsTrue()
			g.Assert(params["foo"] == "super_duper_secret").IsTrue()
		})
	})
}
