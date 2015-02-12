package repo

import (
	"github.com/drone/drone/shared/model"
	"testing"
)

func TestIsRemote(t *testing.T) {
	repo := Repo{Repo: &model.Repo{}}
	if remote := repo.IsRemote(); remote != true {
		t.Errorf("IsRemote with Repo was %v, expected %v", remote, true)
	}
	repo = Repo{}
	if remote := repo.IsRemote(); remote != false {
		t.Errorf("IsRemote without Repo was %v, expected %v", remote, true)
	}
}
