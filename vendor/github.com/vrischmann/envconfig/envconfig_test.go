package envconfig_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/vrischmann/envconfig"
)

func TestParseSimpleConfig(t *testing.T) {
	var conf struct {
		Name string
		Log  struct {
			Path string
		}
	}

	// Go 1.2 and 1.3 don't have os.Unsetenv
	os.Setenv("NAME", "")
	os.Setenv("LOG_PATH", "")

	err := envconfig.Init(&conf)
	equals(t, "envconfig: key NAME not found", err.Error())

	os.Setenv("NAME", "foobar")
	err = envconfig.Init(&conf)
	equals(t, "envconfig: key LOG_PATH not found", err.Error())

	os.Setenv("LOG_PATH", "/var/log/foobar")
	err = envconfig.Init(&conf)
	ok(t, err)

	equals(t, "foobar", conf.Name)
	equals(t, "/var/log/foobar", conf.Log.Path)
}

func TestParseIntegerConfig(t *testing.T) {
	var conf struct {
		Port    int
		Long    uint64
		Version uint8
	}

	timestamp := time.Now().UnixNano()

	os.Setenv("PORT", "80")
	os.Setenv("LONG", fmt.Sprintf("%d", timestamp))
	os.Setenv("VERSION", "2")

	err := envconfig.Init(&conf)
	ok(t, err)

	equals(t, 80, conf.Port)
	equals(t, uint64(timestamp), conf.Long)
	equals(t, uint8(2), conf.Version)
}

func TestParseBoolConfig(t *testing.T) {
	var conf struct {
		DoIt bool
	}

	os.Setenv("DOIT", "true")

	err := envconfig.Init(&conf)
	ok(t, err)
	equals(t, true, conf.DoIt)
}

func TestParseBytesConfig(t *testing.T) {
	var conf struct {
		Data []byte
	}

	os.Setenv("DATA", "Rk9PQkFS")

	err := envconfig.Init(&conf)
	ok(t, err)
	equals(t, []byte("FOOBAR"), conf.Data)
}

func TestParseFloatConfig(t *testing.T) {
	var conf struct {
		Delta  float32
		DeltaV float64
	}

	os.Setenv("DELTA", "0.02")
	os.Setenv("DELTAV", "400.20000000001")

	err := envconfig.Init(&conf)
	ok(t, err)
	equals(t, float32(0.02), conf.Delta)
	equals(t, float64(400.20000000001), conf.DeltaV)
}

func TestParseSliceConfig(t *testing.T) {
	var conf struct {
		Names  []string
		Ports  []int
		Shards []struct {
			Name string
			Addr string
		}
	}

	os.Setenv("NAMES", "foobar,barbaz")
	os.Setenv("PORTS", "900,100")
	os.Setenv("SHARDS", "{foobar,localhost:2929},{barbaz,localhost:2828}")

	err := envconfig.Init(&conf)
	ok(t, err)

	equals(t, 2, len(conf.Names))
	equals(t, "foobar", conf.Names[0])
	equals(t, "barbaz", conf.Names[1])
	equals(t, 2, len(conf.Ports))
	equals(t, 900, conf.Ports[0])
	equals(t, 100, conf.Ports[1])
	equals(t, 2, len(conf.Shards))
	equals(t, "foobar", conf.Shards[0].Name)
	equals(t, "localhost:2929", conf.Shards[0].Addr)
	equals(t, "barbaz", conf.Shards[1].Name)
	equals(t, "localhost:2828", conf.Shards[1].Addr)
}

func TestDurationConfig(t *testing.T) {
	var conf struct {
		Timeout time.Duration
	}

	os.Setenv("TIMEOUT", "1m")

	err := envconfig.Init(&conf)
	ok(t, err)

	equals(t, time.Minute*1, conf.Timeout)
}

func TestInvalidDurationConfig(t *testing.T) {
	var conf struct {
		Timeout time.Duration
	}

	os.Setenv("TIMEOUT", "foo")

	err := envconfig.Init(&conf)
	assert(t, err != nil, "err should not be nil")
}

func TestAllPointerConfig(t *testing.T) {
	var conf struct {
		Name   *string
		Port   *int
		Delta  *float32
		DeltaV *float64
		Hosts  *[]string
		Shards *[]*struct {
			Name *string
			Addr *string
		}
		Master *struct {
			Name *string
			Addr *string
		}
		Timeout *time.Duration
	}

	os.Setenv("NAME", "foobar")
	os.Setenv("PORT", "9000")
	os.Setenv("DELTA", "40.01")
	os.Setenv("DELTAV", "200.00001")
	os.Setenv("HOSTS", "localhost,free.fr")
	os.Setenv("SHARDS", "{foobar,localhost:2828},{barbaz,localhost:2929}")
	os.Setenv("MASTER_NAME", "master")
	os.Setenv("MASTER_ADDR", "localhost:2727")
	os.Setenv("TIMEOUT", "1m")

	err := envconfig.Init(&conf)
	ok(t, err)

	equals(t, "foobar", *conf.Name)
	equals(t, 9000, *conf.Port)
	equals(t, float32(40.01), *conf.Delta)
	equals(t, 200.00001, *conf.DeltaV)
	equals(t, 2, len(*conf.Hosts))
	equals(t, "localhost", (*conf.Hosts)[0])
	equals(t, "free.fr", (*conf.Hosts)[1])
	equals(t, 2, len(*conf.Shards))
	equals(t, "foobar", *(*conf.Shards)[0].Name)
	equals(t, "localhost:2828", *(*conf.Shards)[0].Addr)
	equals(t, "barbaz", *(*conf.Shards)[1].Name)
	equals(t, "localhost:2929", *(*conf.Shards)[1].Addr)
	equals(t, "master", *conf.Master.Name)
	equals(t, "localhost:2727", *conf.Master.Addr)
	equals(t, time.Minute*1, *conf.Timeout)
}

type logMode uint

const (
	logFile logMode = iota + 1
	logStdout
)

func (m *logMode) Unmarshal(s string) error {
	switch strings.ToLower(s) {
	case "file":
		*m = logFile
	case "stdout":
		*m = logStdout
	default:
		return fmt.Errorf("unable to unmarshal %s", s)
	}

	return nil
}

func TestUnmarshaler(t *testing.T) {
	var conf struct {
		LogMode logMode
	}

	os.Setenv("LOGMODE", "file")

	err := envconfig.Init(&conf)
	ok(t, err)

	equals(t, logFile, conf.LogMode)
}

func TestParseOptionalConfig(t *testing.T) {
	var conf struct {
		Name    string        `envconfig:"optional"`
		Flag    bool          `envconfig:"optional"`
		Timeout time.Duration `envconfig:"optional"`
		Port    int           `envconfig:"optional"`
		Port2   uint          `envconfig:"optional"`
		Delta   float32       `envconfig:"optional"`
		DeltaV  float64       `envconfig:"optional"`
		Slice   []string      `envconfig:"optional"`
		Struct  struct {
			A string
			B int
		} `envconfig:"optional"`
	}

	os.Setenv("NAME", "")
	os.Setenv("FLAG", "")
	os.Setenv("TIMEOUT", "")
	os.Setenv("PORT", "")
	os.Setenv("PORT2", "")
	os.Setenv("DELTA", "")
	os.Setenv("DELTAV", "")
	os.Setenv("SLICE", "")
	os.Setenv("STRUCT", "")

	err := envconfig.Init(&conf)
	ok(t, err)
	equals(t, "", conf.Name)
}

func TestParseSkippableConfig(t *testing.T) {
	var conf struct {
		Flag bool `envconfig:"-"`
	}

	os.Setenv("FLAG", "true")

	err := envconfig.Init(&conf)
	ok(t, err)
	equals(t, false, conf.Flag)
}

func TestParseCustomNameConfig(t *testing.T) {
	var conf struct {
		Name string `envconfig:"customName"`
	}

	os.Setenv("customName", "foobar")

	err := envconfig.Init(&conf)
	ok(t, err)
	equals(t, "foobar", conf.Name)
}

func TestParseOptionalStruct(t *testing.T) {
	var conf struct {
		Master struct {
			Name string
		} `envconfig:"optional"`
	}

	os.Setenv("MASTER_NAME", "")

	err := envconfig.Init(&conf)
	ok(t, err)
	equals(t, "", conf.Master.Name)
}

func TestParsePrefixedStruct(t *testing.T) {
	var conf struct {
		Name string
	}

	os.Setenv("NAME", "")
	os.Setenv("FOO_NAME", "")

	os.Setenv("NAME", "bad")
	err := envconfig.InitWithPrefix(&conf, "FOO")
	assert(t, err != nil, "err should not be nil")

	os.Setenv("FOO_NAME", "good")
	err = envconfig.InitWithPrefix(&conf, "FOO")
	ok(t, err)
	equals(t, "good", conf.Name)
}

func TestUnexportedField(t *testing.T) {
	var conf struct {
		name string
	}

	os.Setenv("NAME", "foobar")

	err := envconfig.Init(&conf)
	equals(t, envconfig.ErrUnexportedField, err)
}

// assert fails the test if the condition is false.
func assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

// ok fails the test if an err is not nil.
func ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}

// equals fails the test if exp is not equal to act.
func equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}
