package main

import (
	"os"
	"testing"
)

func TestGetExternalVariables(t *testing.T) {
	pprefix := "DRONE_TEST_GEV_"

	os.Setenv(pprefix + "FOO_BAR", "BAZ")
	os.Setenv(pprefix + "FOOD_BAR", "PIZZA")

	ext := make(map[string]string)
	getExternalVariables(pprefix + "FOO_", ext)

	if len(ext) == 0 {
		t.Error("No external variables found")
	} else if len(ext) > 1 {
		t.Error("Too many external variables found")
	} else if ext["BAR"] != "BAZ" {
		t.Error("Incorrect external variable found")
	}

	// make sure we can overwrite existing variables
	getExternalVariables(pprefix + "FOOD_", ext)
	if len(ext) > 1 {
		t.Error("Too many external variables found")
	} else if ext["BAR"] != "PIZZA" {
		t.Error("Failed to overwrite external variable")
	}
}
