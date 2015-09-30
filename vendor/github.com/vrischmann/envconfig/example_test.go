package envconfig_test

import (
	"fmt"
	"os"
	"time"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/vrischmann/envconfig"
)

func ExampleInit() {
	var conf struct {
		MySQL struct {
			Host     string
			Port     int
			Database struct {
				User     string
				Password string
				Name     string
			}
			Params struct {
				Charset string `envconfig:"-"`
			}
		}
		Log struct {
			Path   string
			Rotate bool
		}
		NbWorkers int
		Timeout   time.Duration
	}

	os.Setenv("MYSQL_HOST", "localhost")
	os.Setenv("MYSQL_PORT", "3306")
	os.Setenv("MYSQL_DATABASE_USER", "root")
	os.Setenv("MYSQL_DATABASE_PASSWORD", "foobar")
	os.Setenv("MYSQL_DATABASE_NAME", "default")
	os.Setenv("LOG_PATH", "/var/log/foobar.log")
	os.Setenv("LOG_ROTATE", "true")
	os.Setenv("NBWORKERS", "10")
	os.Setenv("TIMEOUT", "120s")

	if err := envconfig.Init(&conf); err != nil {
		fmt.Printf("err=%s\n", err)
	}

	fmt.Println(conf.MySQL.Database.User)
	fmt.Println(conf.Log.Rotate)
	fmt.Println(conf.Timeout)
	// Output:
	// root
	// true
	// 2m0s
}

func ExampleInitWithPrefix() {
	var conf struct {
		Name string
	}

	os.Setenv("NAME", "")
	os.Setenv("FOO_NAME", "")

	os.Setenv("NAME", "foobar")

	err := envconfig.InitWithPrefix(&conf, "FOO")
	fmt.Println(err)

	os.Setenv("FOO_NAME", "foobar")
	err = envconfig.InitWithPrefix(&conf, "FOO")
	fmt.Println(err)

	fmt.Println(conf.Name)
	// Output:
	// envconfig: key FOO_NAME not found
	// <nil>
	// foobar
}
