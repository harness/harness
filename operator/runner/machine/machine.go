// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package machine

import (
	"errors"
	"io/ioutil"
	"path/filepath"
)

// ErrNoMachines is returned when no valid or matching
// docker machines are found in the docker-machine home
// directory.
var ErrNoMachines = errors.New("No Docker Machines found")

// Load loads the docker-machine runners.
func Load(home, match string) ([]*Config, error) {
	path := filepath.Join(home, "machines")
	entries, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	// loop through the list of docker-machine home
	// and capture a list of matching subdirectories.
	var machines []*Config
	for _, entry := range entries {
		if entry.IsDir() == false {
			continue
		}
		name := entry.Name()
		confPath := filepath.Join(path, name, "config.json")
		conf, err := parseFile(confPath)
		if err != nil {
			return nil, err
		}
		// If no match logic is defined, the matchine is
		// automatically used as a build machine.
		if match == "" {
			machines = append(machines, conf)
			continue
		}
		// Else verify the machine matches the user-defined
		// pattern. Use as a build machine if a match exists
		match, _ := filepath.Match(match, conf.Name)
		if match {
			machines = append(machines, conf)
		}
	}
	if len(machines) == 0 {
		return nil, ErrNoMachines
	}
	return machines, nil
}
