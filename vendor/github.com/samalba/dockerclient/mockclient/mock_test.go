package mockclient

import (
	"reflect"
	"testing"

	"github.com/samalba/dockerclient"
)

func TestMock(t *testing.T) {
	mock := NewMockClient()
	mock.On("Version").Return(&dockerclient.Version{Version: "foo"}, nil).Once()

	v, err := mock.Version()
	if err != nil {
		t.Fatal(err)
	}
	if v.Version != "foo" {
		t.Fatal(v)
	}

	mock.Mock.AssertExpectations(t)
}

func TestMockInterface(t *testing.T) {
	iface := reflect.TypeOf((*dockerclient.Client)(nil)).Elem()
	mock := NewMockClient()

	if !reflect.TypeOf(mock).Implements(iface) {
		t.Fatalf("Mock does not implement the Client interface")
	}
}
