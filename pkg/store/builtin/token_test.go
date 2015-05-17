package builtin

import (
	"testing"
	"time"

	common "github.com/drone/drone/pkg/types"
	"github.com/franela/goblin"
)

func TestTokenstore(t *testing.T) {
	db := mustConnectTest()
	ts := NewTokenstore(db)
	defer db.Close()

	g := goblin.Goblin(t)
	g.Describe("Tokenstore", func() {

		// before each test be sure to purge the package
		// table data from the database.
		g.BeforeEach(func() {
			db.Exec("DELETE FROM tokens")
		})

		g.It("Should Add a new Token", func() {
			token := common.Token{
				UserID: 1,
				Label:  "foo",
				Kind:   common.TokenUser,
				Issued: time.Now().Unix(),
				Expiry: time.Now().Unix() + 1000,
			}
			err := ts.AddToken(&token)
			g.Assert(err == nil).IsTrue()
			g.Assert(token.ID != 0).IsTrue()
		})

		g.It("Should get a Token", func() {
			token := common.Token{
				UserID: 1,
				Label:  "foo",
				Kind:   common.TokenUser,
				Issued: time.Now().Unix(),
				Expiry: time.Now().Unix() + 1000,
			}
			err1 := ts.AddToken(&token)
			gettoken, err2 := ts.Token(token.ID)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(token.ID).Equal(gettoken.ID)
			g.Assert(token.Label).Equal(gettoken.Label)
			g.Assert(token.Kind).Equal(gettoken.Kind)
			g.Assert(token.Issued).Equal(gettoken.Issued)
			g.Assert(token.Expiry).Equal(gettoken.Expiry)
		})

		g.It("Should Get a Token By Label", func() {
			token := common.Token{
				UserID: 1,
				Label:  "foo",
				Kind:   common.TokenUser,
				Issued: time.Now().Unix(),
				Expiry: time.Now().Unix() + 1000,
			}
			err1 := ts.AddToken(&token)
			gettoken, err2 := ts.TokenLabel(&common.User{ID: 1}, "foo")
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(token.ID).Equal(gettoken.ID)
			g.Assert(token.Label).Equal(gettoken.Label)
			g.Assert(token.Kind).Equal(gettoken.Kind)
			g.Assert(token.Issued).Equal(gettoken.Issued)
			g.Assert(token.Expiry).Equal(gettoken.Expiry)
		})

		g.It("Should Enforce Unique Token Label", func() {
			token1 := common.Token{
				UserID: 1,
				Label:  "foo",
				Kind:   common.TokenUser,
				Issued: time.Now().Unix(),
				Expiry: time.Now().Unix() + 1000,
			}
			token2 := common.Token{
				UserID: 1,
				Label:  "foo",
				Kind:   common.TokenUser,
				Issued: time.Now().Unix(),
				Expiry: time.Now().Unix() + 1000,
			}
			err1 := ts.AddToken(&token1)
			err2 := ts.AddToken(&token2)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsFalse()
		})

		g.It("Should Get a User Token List", func() {
			token1 := common.Token{
				UserID: 1,
				Label:  "bar",
				Kind:   common.TokenUser,
				Issued: time.Now().Unix(),
				Expiry: time.Now().Unix() + 1000,
			}
			token2 := common.Token{
				UserID: 1,
				Label:  "foo",
				Kind:   common.TokenUser,
				Issued: time.Now().Unix(),
				Expiry: time.Now().Unix() + 1000,
			}
			token3 := common.Token{
				UserID: 2,
				Label:  "foo",
				Kind:   common.TokenUser,
				Issued: time.Now().Unix(),
				Expiry: time.Now().Unix() + 1000,
			}
			ts.AddToken(&token1)
			ts.AddToken(&token2)
			ts.AddToken(&token3)
			tokens, err := ts.TokenList(&common.User{ID: 1})
			g.Assert(err == nil).IsTrue()
			g.Assert(len(tokens)).Equal(2)
			g.Assert(tokens[0].ID).Equal(token1.ID)
			g.Assert(tokens[0].Label).Equal(token1.Label)
			g.Assert(tokens[0].Kind).Equal(token1.Kind)
			g.Assert(tokens[0].Issued).Equal(token1.Issued)
			g.Assert(tokens[0].Expiry).Equal(token1.Expiry)
		})

		g.It("Should Del a Token", func() {
			token := common.Token{
				UserID: 1,
				Label:  "foo",
				Kind:   common.TokenUser,
				Issued: time.Now().Unix(),
				Expiry: time.Now().Unix() + 1000,
			}
			ts.AddToken(&token)
			_, err1 := ts.Token(token.ID)
			err2 := ts.DelToken(&token)
			_, err3 := ts.Token(token.ID)
			g.Assert(err1 == nil).IsTrue()
			g.Assert(err2 == nil).IsTrue()
			g.Assert(err3 == nil).IsFalse()
		})
	})
}
