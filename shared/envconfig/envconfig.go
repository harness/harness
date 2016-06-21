package envconfig

import (
	"bufio"
	"errors"
	log "github.com/Sirupsen/logrus"
	"os"
	"strconv"
	"strings"
)

type Env map[string]string

var envRequired = []string{"SERVER_ADDR", "REMOTE_DRIVER", "REMOTE_CONFIG", "RC_SRY_REG_INSECURE",
	"RC_SRY_REG_HOST", "PUBLIC_MODE", "DATABASE_DRIVER", "DATABASE_CONFIG",
	"AGENT_URI", "PLUGIN_FILTER", "PLUGIN_PREFIX", "DOCKER_STORAGE", "DOCKER_EXTRA_HOSTS"}

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
			if len(os.Getenv(strings.ToUpper(key))) == 0 {
				os.Setenv(strings.ToUpper(key), val)
			}
			//	os.Setenv(key, val)
		}
	}

	//check required env
	for _, entry := range envRequired {
		if len(os.Getenv(entry)) == 0 {
			exitMissingEnv(entry)
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
func exitMissingEnv(env string) {
	log.Errorf("program exit missing config for env %s", env)
	os.Exit(1)
}
