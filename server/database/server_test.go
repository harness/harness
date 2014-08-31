package database

import (
	"testing"

	"github.com/drone/drone/shared/model"
	"github.com/jinzhu/gorm"
)

func TestServerFind(t *testing.T) {
	setup()
	defer teardown()

	servers := NewServerManager(conn.DB)
	server, err := servers.Find(1)
	if err != nil {
		t.Errorf("Want Server from ID, got %s", err)
	}

	testServer(t, server)
}

func TestServerFindName(t *testing.T) {
	setup()
	defer teardown()

	servers := NewServerManager(conn.DB)
	server, err := servers.FindName("docker1")
	if err != nil {
		t.Errorf("Want Server from Host, got %s", err)
	}

	testServer(t, server)
}

func TestServerFindSMTP(t *testing.T) {
	setup()
	defer teardown()
	server := model.SMTPServer{
		From: "foo@bar.com",
		Host: "127.0.0.1",
		User: "foo"}

	servers := NewServerManager(conn.DB)
	if err := servers.UpdateSMTP(&server); err != nil {
		t.Errorf("Want SMTP server inserted, got %s", err)
	}
	if inserted, err := servers.FindSMTP(); err != nil {
		t.Errorf("Want SMTP server, got %s", err)
	} else if inserted.Id == 0 {
		t.Errorf("Want SMTP server inserted")
	}

	server.Host = "0.0.0.0"
	server.User = "bar"
	err := servers.UpdateSMTP(&server)
	if err := servers.UpdateSMTP(&server); err != nil {
		t.Errorf("Want SMTP server updated, got %s", err)
	}

	updated, err := servers.FindSMTP()
	if err != nil {
		t.Errorf("Want SMTP server, got %s", err)
	}

	var want, got = server.Host, updated.Host
	if want != got {
		t.Errorf("Want SMTP Host %v, got %v", want, got)
	}
}

func TestServerList(t *testing.T) {
	setup()
	defer teardown()

	servers := NewServerManager(conn.DB)
	all, err := servers.List()
	if err != nil {
		t.Errorf("Want Servers, got %s", err)
	}

	var got, want = len(all), 2
	if got != want {
		t.Errorf("Want %v Servers, got %v", want, got)
	}

	testServer(t, all[0])
}

func TestServerInsert(t *testing.T) {
	setup()
	defer teardown()

	server := &model.Server{Host: "tcp://127.0.0.1:4243", Name: "docker3"}
	servers := NewServerManager(conn.DB)
	if err := servers.Insert(server); err != nil {
		t.Errorf("Want Server created, got %s", err)
	}

	var got, want = server.Id, int64(3)
	if want != got {
		t.Errorf("Want Server ID %v, got %v", want, got)
	}

	// verify unique server name constraint
	var err = servers.Insert(&model.Server{Host: "tcp://127.0.0.1:4243", Name: "docker3"})
	if err == nil {
		t.Error("Want Name unique constraint violated")
	}
}

func TestServerUpdate(t *testing.T) {
	setup()
	defer teardown()

	servers := NewServerManager(conn.DB)
	server, err := servers.Find(1)
	if err != nil {
		t.Errorf("Want Server from ID, got %s", err)
	}

	// update the server's address
	server.Host = "tcp://1.2.3.4:4243"
	server.User = "docker"
	server.Pass = "123456"
	if err := servers.Update(server); err != nil {
		t.Errorf("Want Server updated, got %s", err)
	}

	updated, _ := servers.Find(1)
	var got, want = server.Host, server.Host
	if got != want {
		t.Errorf("Want updated Host %s, got %s", want, got)
	}

	got, want = updated.User, server.User
	if got != want {
		t.Errorf("Want updated User %s, got %s", want, got)
	}

	got, want = updated.Pass, server.Pass
	if got != want {
		t.Errorf("Want updated Pass %s, got %s", want, got)
	}
}

func TestServerDelete(t *testing.T) {
	setup()
	defer teardown()

	servers := NewServerManager(conn.DB)
	server, err := servers.Find(1)
	if err != nil {
		t.Errorf("Want Server from ID, got %s", err)
	}

	// delete the server
	if err := servers.Delete(server); err != nil {
		t.Errorf("Want Server deleted, got %s", err)
	}

	// check to see if the deleted server is actually gone
	if _, err := servers.Find(1); err != gorm.RecordNotFound {
		t.Errorf("Want ErrNoRows, got %s", err)
	}
}

// testServer is a helper function that compares the server
// to an expected set of fixed field values.
func testServer(t *testing.T, server *model.Server) {

	var got, want = server.Host, "tcp://127.0.0.1:4243"
	if got != want {
		t.Errorf("Want Host %v, got %v", want, got)
	}

	got, want = server.Name, "docker1"
	if got != want {
		t.Errorf("Want Name %v, got %v", want, got)
	}

	got, want = server.User, "root"
	if got != want {
		t.Errorf("Want User %v, got %v", want, got)
	}

	got, want = server.Pass, "pa55word"
	if got != want {
		t.Errorf("Want Pass %v, got %v", want, got)
	}

	got, want = server.Cert, "/path/to/cert.key"
	if got != want {
		t.Errorf("Want Cert %v, got %v", want, got)
	}

	var gotInt64, wantInt64 = server.Id, int64(1)
	if gotInt64 != wantInt64 {
		t.Errorf("Want ID %v, got %v", wantInt64, gotInt64)
	}
}
