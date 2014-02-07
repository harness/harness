package database

import (
	"testing"

	"github.com/drone/drone/pkg/database"
)

func TestGetCommit(t *testing.T) {
	Setup()
	defer Teardown()

	commit, err := database.GetCommit(1)
	if err != nil {
		t.Error(err)
	}

	if commit.ID != 1 {
		t.Errorf("Exepected ID %d, got %d", 1, commit.ID)
	}

	if commit.Status != "Success" {
		t.Errorf("Exepected Status %s, got %s", "Success", commit.Status)
	}

	if commit.Hash != "4f4c4594be6d6ddbc1c0dd521334f7ecba92b608" {
		t.Errorf("Exepected Hash %s, got %s", "4f4c4594be6d6ddbc1c0dd521334f7ecba92b608", commit.Hash)
	}

	if commit.Branch != "master" {
		t.Errorf("Exepected Branch %s, got %s", "master", commit.Branch)
	}

	if commit.Author != "brad.rydzewski@gmail.com" {
		t.Errorf("Exepected Author %s, got %s", "master", commit.Author)
	}

	if commit.Message != "commit message" {
		t.Errorf("Exepected Message %s, got %s", "master", commit.Message)
	}

	if commit.Gravatar != "8c58a0be77ee441bb8f8595b7f1b4e87" {
		t.Errorf("Exepected Gravatar %s, got %s", "8c58a0be77ee441bb8f8595b7f1b4e87", commit.Gravatar)
	}
}

func TestGetCommitHash(t *testing.T) {
	Setup()
	defer Teardown()

	commit, err := database.GetCommitHash("4f4c4594be6d6ddbc1c0dd521334f7ecba92b608", 1)
	if err != nil {
		t.Error(err)
	}

	if commit.ID != 1 {
		t.Errorf("Exepected ID %d, got %d", 1, commit.ID)
	}

	if commit.Hash != "4f4c4594be6d6ddbc1c0dd521334f7ecba92b608" {
		t.Errorf("Exepected Hash %s, got %s", "4f4c4594be6d6ddbc1c0dd521334f7ecba92b608", commit.Hash)
	}

	if commit.Status != "Success" {
		t.Errorf("Exepected Status %s, got %s", "Success", commit.Status)
	}
}

func TestSaveCommit(t *testing.T) {
	Setup()
	defer Teardown()

	// get the commit we plan to update
	commit, err := database.GetCommit(1)
	if err != nil {
		t.Error(err)
	}

	// update fields
	commit.Status = "Failing"

	// update the database
	if err := database.SaveCommit(commit); err != nil {
		t.Error(err)
	}

	// get the updated commit
	updatedCommit, err := database.GetCommit(1)
	if err != nil {
		t.Error(err)
	}

	if commit.Hash != updatedCommit.Hash {
		t.Errorf("Exepected Hash %s, got %s", updatedCommit.Hash, commit.Hash)
	}

	if commit.Status != "Failing" {
		t.Errorf("Exepected Status %s, got %s", updatedCommit.Status, commit.Status)
	}
}

func TestDeleteCommit(t *testing.T) {
	Setup()
	defer Teardown()

	if err := database.DeleteCommit(1); err != nil {
		t.Error(err)
	}

	// try to get the deleted row
	_, err := database.GetCommit(1)
	if err == nil {
		t.Fail()
	}
}

func TestListCommits(t *testing.T) {
	Setup()
	defer Teardown()

	// commits for repo_id = 1
	commits, err := database.ListCommits(1, "master")
	if err != nil {
		t.Error(err)
	}

	// verify commit count
	if len(commits) != 2 {
		t.Errorf("Exepected %d commits in database, got %d", 2, len(commits))
		return
	}

	// get the first user in the list and verify
	// fields are being populated correctly
	commit := commits[1] // TODO something strange is happening with ordering here

	if commit.ID != 1 {
		t.Errorf("Exepected ID %d, got %d", 1, commit.ID)
	}

	if commit.Status != "Success" {
		t.Errorf("Exepected Status %s, got %s", "Success", commit.Status)
	}

	if commit.Hash != "4f4c4594be6d6ddbc1c0dd521334f7ecba92b608" {
		t.Errorf("Exepected Hash %s, got %s", "4f4c4594be6d6ddbc1c0dd521334f7ecba92b608", commit.Hash)
	}

	if commit.Branch != "master" {
		t.Errorf("Exepected Branch %s, got %s", "master", commit.Branch)
	}

	if commit.Author != "brad.rydzewski@gmail.com" {
		t.Errorf("Exepected Author %s, got %s", "master", commit.Author)
	}

	if commit.Message != "commit message" {
		t.Errorf("Exepected Message %s, got %s", "master", commit.Message)
	}

	if commit.Gravatar != "8c58a0be77ee441bb8f8595b7f1b4e87" {
		t.Errorf("Exepected Gravatar %s, got %s", "8c58a0be77ee441bb8f8595b7f1b4e87", commit.Gravatar)
	}
}
