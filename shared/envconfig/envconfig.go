package envconfig

import (
	"bufio"
	"errors"
	"os"
	"strconv"
	"strings"
)

type Env map[string]string

// Get returns the value of the environment variable named by the key.
func (env Env) Get(key string) string {
	return env[key]
}

// String returns the string value of the environment variable named by the
// key. If the variable is not present, the default value is returned.
func (env Env) String(key, value string) string {
	got, ok := env[key]
	if ok {
		value = got
	}
	return value
}

// Bool returns the boolean value of the environment variable named by the key.
// If the variable is not present, the default value is returned.
func (env Env) Bool(name string, value bool) bool {
	got, ok := env[name]
	if ok {
		value, _ = strconv.ParseBool(got)
	}
	return value
}

// Int returns the integer value of the environment variable named by the key.
// If the variable is not present, the default value is returned.
func (env Env) Int(name string, value int) int {
	got, ok := env[name]
	if ok {
		value, _ = strconv.Atoi(got)
	}
	return value
}

// Load reads the environment file and reads variables in "key=value" format.
// Then it read the system environment variables. It returns the combined
// results in a key value map.
func Load(filepath string) Env {
	var envs = map[string]string{}

	// load the environment file
	f, err := os.Open(filepath)
	if err == nil {
		defer f.Close()

		r := bufio.NewReader(f)
		for {
			line, _, err := r.ReadLine()
			if err != nil {
				break
			}

			key, val, err := parseln(string(line))
			if err != nil {
				continue
			}

			os.Setenv(key, val)
		}
	}

	// load the environment variables
	for _, env := range os.Environ() {
		key, val, err := parseln(env)
		if err != nil {
			continue
		}

		envs[key] = val
	}

	return Env(envs)
}

// helper function to parse a "key=value" environment variable string.
func parseln(line string) (key string, val string, err error) {
	line = removeComments(line)
	if len(line) == 0 {
		return
	}
	splits := strings.SplitN(line, "=", 2)

	if len(splits) < 2 {
		err = errors.New("missing delimiter '='")
		return
	}

	key = strings.Trim(splits[0], " ")
	val = strings.Trim(splits[1], ` "'`)
	return
}

// helper function to trim comments and whitespace from a string.
func removeComments(s string) (_ string) {
	if len(s) == 0 || string(s[0]) == "#" {
		return
	} else {
		index := strings.Index(s, " #")
		if index > -1 {
			s = strings.TrimSpace(s[0:index])
		}
	}
	return s
}
