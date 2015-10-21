package datastore

import (
	"testing"

	"github.com/drone/drone/model"
	"github.com/franela/goblin"
)

func Test_keystore(t *testing.T) {
	db := openTest()
	defer db.Close()

	s := From(db)
	g := goblin.Goblin(t)
	g.Describe("Keys", func() {

		// before each test be sure to purge the package
		// table data from the database.
		g.BeforeEach(func() {
			db.Exec(rebind("DELETE FROM `keys`"))
		})

		g.It("Should create a key", func() {
			key := model.Key{
				RepoID:  1,
				Public:  fakePublicKey,
				Private: fakePrivateKey,
			}
			err := s.Keys().Create(&key)
			g.Assert(err == nil).IsTrue()
			g.Assert(key.ID != 0).IsTrue()
		})

		g.It("Should update a key", func() {
			key := model.Key{
				RepoID:  1,
				Public:  fakePublicKey,
				Private: fakePrivateKey,
			}
			err := s.Keys().Create(&key)
			g.Assert(err == nil).IsTrue()
			g.Assert(key.ID != 0).IsTrue()

			key.Private = ""
			key.Public = ""

			err1 := s.Keys().Update(&key)
			getkey, err2 := s.Keys().Get(&model.Repo{ID: 1})
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(key.ID).Equal(getkey.ID)
			g.Assert(key.Public).Equal(getkey.Public)
			g.Assert(key.Private).Equal(getkey.Private)
		})

		g.It("Should get a key", func() {
			key := model.Key{
				RepoID:  1,
				Public:  fakePublicKey,
				Private: fakePrivateKey,
			}
			err := s.Keys().Create(&key)
			g.Assert(err == nil).IsTrue()
			g.Assert(key.ID != 0).IsTrue()

			getkey, err := s.Keys().Get(&model.Repo{ID: 1})
			g.Assert(err == nil).IsTrue()
			g.Assert(key.ID).Equal(getkey.ID)
			g.Assert(key.Public).Equal(getkey.Public)
			g.Assert(key.Private).Equal(getkey.Private)
		})

		g.It("Should delete a key", func() {
			key := model.Key{
				RepoID:  1,
				Public:  fakePublicKey,
				Private: fakePrivateKey,
			}
			err1 := s.Keys().Create(&key)
			err2 := s.Keys().Delete(&key)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()

			_, err := s.Keys().Get(&model.Repo{ID: 1})
			g.Assert(err == nil).IsFalse()
		})
	})
}

var fakePublicKey = `
-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCqGKukO1De7zhZj6+H0qtjTkVxwTCpvKe4eCZ0
FPqri0cb2JZfXJ/DgYSF6vUpwmJG8wVQZKjeGcjDOL5UlsuusFncCzWBQ7RKNUSesmQRMSGkVb1/
3j+skZ6UtW+5u09lHNsj6tQ51s1SPrCBkedbNf0Tp0GbMJDyR4e9T04ZZwIDAQAB
-----END PUBLIC KEY-----
`

var fakePrivateKey = `

-----BEGIN RSA PRIVATE KEY-----
MIICXAIBAAKBgQCqGKukO1De7zhZj6+H0qtjTkVxwTCpvKe4eCZ0FPqri0cb2JZfXJ/DgYSF6vUp
wmJG8wVQZKjeGcjDOL5UlsuusFncCzWBQ7RKNUSesmQRMSGkVb1/3j+skZ6UtW+5u09lHNsj6tQ5
1s1SPrCBkedbNf0Tp0GbMJDyR4e9T04ZZwIDAQABAoGAFijko56+qGyN8M0RVyaRAXz++xTqHBLh
3tx4VgMtrQ+WEgCjhoTwo23KMBAuJGSYnRmoBZM3lMfTKevIkAidPExvYCdm5dYq3XToLkkLv5L2
pIIVOFMDG+KESnAFV7l2c+cnzRMW0+b6f8mR1CJzZuxVLL6Q02fvLi55/mbSYxECQQDeAw6fiIQX
GukBI4eMZZt4nscy2o12KyYner3VpoeE+Np2q+Z3pvAMd/aNzQ/W9WaI+NRfcxUJrmfPwIGm63il
AkEAxCL5HQb2bQr4ByorcMWm/hEP2MZzROV73yF41hPsRC9m66KrheO9HPTJuo3/9s5p+sqGxOlF
L0NDt4SkosjgGwJAFklyR1uZ/wPJjj611cdBcztlPdqoxssQGnh85BzCj/u3WqBpE2vjvyyvyI5k
X6zk7S0ljKtt2jny2+00VsBerQJBAJGC1Mg5Oydo5NwD6BiROrPxGo2bpTbu/fhrT8ebHkTz2epl
U9VQQSQzY1oZMVX8i1m5WUTLPz2yLJIBQVdXqhMCQBGoiuSoSjafUhV7i1cEGpb88h5NBYZzWXGZ
37sJ5QsW+sJyoNde3xH8vdXhzU7eT82D6X/scw9RZz+/6rCJ4p0=
-----END RSA PRIVATE KEY-----
`
