package bolt

import (
	"github.com/drone/drone/common"
	"github.com/drone/drone/common/sshutil"
	"testing"
	//. "github.com/franela/goblin"
	. "github.com/smartystreets/goconvey/convey"
)

func TestRepo(t *testing.T) {
	/*g := Goblin(t)
	g.Describe("Repos", func() {

		g.It("Should find by name") // ok
		g.It("Should find params")
		g.It("Should find keys")
		g.It("Should delete") // ok
		g.It("Should insert") // ok
		g.It("Should not insert if exists")  // ?
		g.It("Should insert params")
		g.It("Should update params")
		g.It("Should insert keys")
		g.It("Should update keys")
	})
	*/
	var testUser string = "octocat"
	var testRepo string = "github.com/octopod/hq"
	var db *DB //-- Temp database
	testParamsIns := map[string]string{
		"A": "Alpha",
		"B": "Beta",
		"C": "Charlie",
		"D": "Delta",
		"E": "Echo",
	}
	testParamsUpd := map[string]string{
		"A": "Alpha-Upd",
		"B": "Beta-Upd",
		"C": "Charlie-Upd",
		"D": "Delta-Upd",
		"E": "Echo-Upd",
	}

	//--
	db = Must("/tmp/drone.test.db")

	//--
	Convey("Should insert", t, func() {
		err := db.InsertRepo(testUser, testRepo)
		So(err, ShouldBeNil)
		//So(InsertRepo(testUser, testRepo), ShouldE)
	})

	//--
	Convey("Should find by name", t, func() {
		repo, err := db.GetRepo(testRepo)
		So(repo, ShoudNotBeNil)
		So(err, ShouldBeNil)
	})

	//-- Lets try to add the same repo again.
	Convey("Should not insert if exists", t, func() {
		err := db.InsertRepo(testUser, testRepo)
		So(err, ShouldEqual, ErrKeyExists)
	})

	//--
	Convey("Should find params", t, func() {
		params, err := db.GetRepoParams(testRepo)
		So(params, ShouldNotBeEmpty)
		So(err, ShouldBeNil)
	})

	//--
	Convey("Should find keys", t, func() {
		keypair, err := db.GetRepoKeys(testRepo)
		So(keypair, ShouldNotBeEmpty)
		So(err, ShouldBeNil)
	})

	//--
	Convey("Should insert params", t, func() {
		err := db.UpsertRepoParams(testRepo, testParamsIns)
		So(err, ShouldBeNil)
	})

	//--
	Convey("Should update params", t, func() {
		err := db.UpsertRepoParams(testRepo, testParamsUpd)
		So(err, ShouldBeNil)
	})

	//--
	Convey("Should insert keys", t, func() {
		//-- generate an RSA key and add to the repo
		testKey, err := sshutil.GeneratePrivateKey()
		So(err, ShouldBeNil)
		//--
		testKeypair := &common.Keypair{}
		testKeypair.Public = sshutil.MarshalPublicKey(&testKey.PublicKey)
		testKeypair.Private = sshutil.MarshalPrivateKey(testKey)
		//--
		err := db.UpsertRepoKeys(testRepo, testKeypair)
		So(err, ShouldBeNil)
	})

	//--
	Convey("Should update keys", t, func() {
		//-- generate an RSA key and add to the repo
		testKey, err := sshutil.GeneratePrivateKey()
		So(err, ShouldBeNil)
		//--
		testKeypair := &common.Keypair{}
		testKeypair.Public = sshutil.MarshalPublicKey(&testKey.PublicKey)
		testKeypair.Private = sshutil.MarshalPrivateKey(testKey)
		//--
		err := db.UpsertRepoKeys(testRepo, testKeypair)
		So(err, ShouldBeNil)
	})

	//--
	Convey("Should delete", t, func() {
		err := db.DeleteRepo(testRepo)
		So(err, ShouldBeNil)
	})

	//-- Delete the temp db at the end.
	os.Remove(db.Path())
}


