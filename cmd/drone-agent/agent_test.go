package main

import (
	"github.com/cncd/pipeline/pipeline/backend"
	"testing"
)

func TestGetRepoName(t *testing.T) {

	backendConfig := new(backend.Config)
	name, err := extractRepositoryName(backendConfig)

	if err == nil {
		t.Errorf("Should return error but instead returned %s", name)
	}

	if name != "" {
		t.Errorf("Should have an empty string as the name.")
	}

	backendConfig.Stages = append(backendConfig.Stages, new(backend.Stage))

	name, err = extractRepositoryName(backendConfig)

	if err == nil {
		t.Errorf("Should return error by instead returned %s", name)
	}

	if name != "" {
		t.Errorf("Should have an empty string as the name.")
	}

	backendConfig.Stages = append(backendConfig.Stages, new(backend.Stage))
	backendConfig.Stages[0].Steps = append(backendConfig.Stages[0].Steps, new(backend.Step))
	backendConfig.Stages[0].Steps[0].Environment = make(map[string]string)

	backendConfig.Stages[0].Steps[0].Environment["DRONE_REPO"] = "TestRepo"

	name, err = extractRepositoryName(backendConfig)

	if err != nil {
		t.Errorf("Should not return error.")
	}

	if name == "" {
		t.Errorf("Should not have an empty string as the name.")
	}

	if name != "TestRepo" {
		t.Errorf("Repo name should match environment variable DRONE_REPO.")
	}
}
