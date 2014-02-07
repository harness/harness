package database

import (
	"testing"

	"github.com/drone/drone/pkg/database"
)

func TestGetBuild(t *testing.T) {
	Setup()
	defer Teardown()

	build, err := database.GetBuild(1)
	if err != nil {
		t.Error(err)
	}

	if build.ID != 1 {
		t.Errorf("Exepected ID %d, got %d", 1, build.ID)
	}

	if build.Slug != "node_0.10" {
		t.Errorf("Exepected Slug %s, got %s", "node_0.10", build.Slug)
	}

	if build.Status != "Success" {
		t.Errorf("Exepected Status %s, got %s", "Success", build.Status)
	}
}

func TestGetBuildSlug(t *testing.T) {
	Setup()
	defer Teardown()

	build, err := database.GetBuildSlug("node_0.10", 1)
	if err != nil {
		t.Error(err)
	}

	if build.ID != 1 {
		t.Errorf("Exepected ID %d, got %d", 1, build.ID)
	}

	if build.Slug != "node_0.10" {
		t.Errorf("Exepected Slug %s, got %s", "node_0.10", build.Slug)
	}

	if build.Status != "Success" {
		t.Errorf("Exepected Status %s, got %s", "Success", build.Status)
	}
}

func TestSaveBbuild(t *testing.T) {
	Setup()
	defer Teardown()

	// get the build we plan to update
	build, err := database.GetBuild(1)
	if err != nil {
		t.Error(err)
	}

	// update fields
	build.Status = "Failing"

	// update the database
	if err := database.SaveBuild(build); err != nil {
		t.Error(err)
	}

	// get the updated build
	updatedBuild, err := database.GetBuild(1)
	if err != nil {
		t.Error(err)
	}

	if build.ID != updatedBuild.ID {
		t.Errorf("Exepected ID %d, got %d", updatedBuild.ID, build.ID)
	}

	if build.Slug != updatedBuild.Slug {
		t.Errorf("Exepected Slug %s, got %s", updatedBuild.Slug, build.Slug)
	}

	if build.Status != updatedBuild.Status {
		t.Errorf("Exepected Status %s, got %s", updatedBuild.Status, build.Status)
	}
}

func TestDeleteBuild(t *testing.T) {
	Setup()
	defer Teardown()

	if err := database.DeleteBuild(1); err != nil {
		t.Error(err)
	}

	// try to get the deleted row
	_, err := database.GetBuild(1)
	if err == nil {
		t.Fail()
	}
}

func TestListBuilds(t *testing.T) {
	Setup()
	defer Teardown()

	// builds for commit_id = 1
	builds, err := database.ListBuilds(1)
	if err != nil {
		t.Error(err)
	}

	// verify user count
	if len(builds) != 2 {
		t.Errorf("Exepected %d builds in database, got %d", 2, len(builds))
		return
	}

	// get the first user in the list and verify
	// fields are being populated correctly
	build := builds[1]

	if build.ID != 1 {
		t.Errorf("Exepected ID %d, got %d", 1, build.ID)
	}

	if build.Slug != "node_0.10" {
		t.Errorf("Exepected Slug %s, got %s", "node_0.10", build.Slug)
	}

	if build.Status != "Success" {
		t.Errorf("Exepected Status %s, got %s", "Success", build.Status)
	}
}
