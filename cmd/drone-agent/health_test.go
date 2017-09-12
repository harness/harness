package main

import (
	"testing"
	"time"
)

func TestHealthy(t *testing.T) {
	s := state{}
	s.Metadata = map[string]info{}

	s.Add("1", time.Hour, "octocat/hello-world", "42")

	if got, want := s.Metadata["1"].ID, "1"; got != want {
		t.Errorf("got ID %s, want %s", got, want)
	}
	if got, want := s.Metadata["1"].Timeout, time.Hour; got != want {
		t.Errorf("got duration %v, want %v", got, want)
	}
	if got, want := s.Metadata["1"].Repo, "octocat/hello-world"; got != want {
		t.Errorf("got repository name %s, want %s", got, want)
	}

	s.Metadata["1"] = info{
		Timeout: time.Hour,
		Started: time.Now().UTC(),
	}
	if s.Healthy() == false {
		t.Error("want healthy status when timeout not exceeded, got false")
	}

	s.Metadata["1"] = info{
		Started: time.Now().UTC().Add(-(time.Minute * 30)),
	}
	if s.Healthy() == false {
		t.Error("want healthy status when timeout+buffer not exceeded, got false")
	}

	s.Metadata["1"] = info{
		Started: time.Now().UTC().Add(-(time.Hour + time.Minute)),
	}
	if s.Healthy() == true {
		t.Error("want unhealthy status when timeout+buffer not exceeded, got true")
	}
}
