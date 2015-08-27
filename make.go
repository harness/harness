// +build ignore

// This program builds Drone.
// $ go run make.go build test
//
// The output binaries go into the ./bin/ directory (under the
// project root, where make.go is)
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jteeuwen/go-bindata"
)

var (
	version = "0.4"
	sha     = rev()
)

// list of all posible steps that can be executed
// as part of the build process.
var steps = map[string]step{
	"scripts": scripts,
	"styles":  styles,
	"json":    json,
	"embed":   embed,
	"vet":     vet,
	"bindata": bindat,
	"build":   build,
	"test":    test,
	"image":   image,
	"clean":   clean,
}

func main() {
	for _, arg := range os.Args[1:] {
		step, ok := steps[arg]
		if !ok {
			fmt.Println("error: invalid step", arg)
			os.Exit(1)
		}
		err := step()
		if err != nil {
			fmt.Println("error: failed step", arg)
			os.Exit(1)
		}
	}
}

type step func() error

// embed step embeds static files in .go files.
func embed() error {
	// embed drone.{revision}.css
	// embed drone.{revision}.js
	return nil
}

// scripts step concatinates all javascript files.
func scripts() error {
	files := []string{
		"cmd/drone-server/static/scripts/term.js",
		"cmd/drone-server/static/scripts/drone.js",
		"cmd/drone-server/static/scripts/controllers/repos.js",
		"cmd/drone-server/static/scripts/controllers/builds.js",
		"cmd/drone-server/static/scripts/controllers/users.js",
		"cmd/drone-server/static/scripts/services/repos.js",
		"cmd/drone-server/static/scripts/services/builds.js",
		"cmd/drone-server/static/scripts/services/users.js",
		"cmd/drone-server/static/scripts/services/logs.js",
		"cmd/drone-server/static/scripts/services/tokens.js",
		"cmd/drone-server/static/scripts/services/feed.js",
		"cmd/drone-server/static/scripts/filters/filter.js",
		"cmd/drone-server/static/scripts/filters/gravatar.js",
		"cmd/drone-server/static/scripts/filters/time.js",
	}

	f, err := os.OpenFile(
		"cmd/drone-server/static/scripts/drone.min.js",
		os.O_CREATE|os.O_RDWR|os.O_TRUNC,
		0660)

	defer f.Close()

	if err != nil {
		fmt.Println("Failed to open output file")
		return err
	}

	for _, input := range files {
		content, err := ioutil.ReadFile(input)

		if err != nil {
			return err
		}

		f.Write(content)
	}

	return nil
}

// styles step concatinates the stylesheet files.
func styles() error {
	files := []string{
		"cmd/drone-server/static/styles/reset.css",
		"cmd/drone-server/static/styles/fonts.css",
		"cmd/drone-server/static/styles/alert.css",
		"cmd/drone-server/static/styles/blankslate.css",
		"cmd/drone-server/static/styles/list.css",
		"cmd/drone-server/static/styles/label.css",
		"cmd/drone-server/static/styles/range.css",
		"cmd/drone-server/static/styles/switch.css",
		"cmd/drone-server/static/styles/main.css",
	}

	f, err := os.OpenFile(
		"cmd/drone-server/static/styles/drone.min.css",
		os.O_CREATE|os.O_RDWR|os.O_TRUNC,
		0660)

	defer f.Close()

	if err != nil {
		fmt.Println("Failed to open output file")
		return err
	}

	for _, input := range files {
		content, err := ioutil.ReadFile(input)

		if err != nil {
			return err
		}

		f.Write(content)
	}

	return nil
}

// json step generates optimized json marshal and
// unmarshal functions to override defaults.
func json() error {
	return nil
}

// bindata step generates go-bindata package.
func bindat() error {
	var paths = []struct {
		input     string
		recursive bool
	}{
		{"cmd/drone-server/static", true},
	}

	c := bindata.NewConfig()
	c.Output = "cmd/drone-server/drone_bindata.go"
	c.Input = make([]bindata.InputConfig, len(paths))

	for i, path := range paths {
		c.Input[i] = bindata.InputConfig{
			Path:      path.input,
			Recursive: path.recursive,
		}
	}

	return bindata.Translate(c)
}

// build step creates the application binaries.
func build() error {
	var bins = []struct {
		input  string
		output string
	}{
		{"github.com/drone/drone/cmd/drone-server", "bin/drone"},
		{"github.com/drone/drone/cmd/drone-agent", "bin/drone-agent"},
		{"github.com/drone/drone/cmd/drone-build", "bin/drone-build"},
	}
	for _, bin := range bins {
		ldf := fmt.Sprintf("-X main.revision=%s -X main.version=%s", sha, version)
		cmd := exec.Command("go", "build", "-o", bin.output, "-ldflags", ldf, bin.input)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		trace(cmd.Args)
		err := cmd.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

// vet step executes the `go vet` command
func vet() error {
	cmd := exec.Command("go", "vet",
		"github.com/drone/drone/pkg/...",
		"github.com/drone/drone/cmd/...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	trace(cmd.Args)
	return cmd.Run()
}

// test step executes unit tests and coverage.
func test() error {
	cmd := exec.Command("go", "test", "-cover", "./pkg/...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	trace(cmd.Args)
	return cmd.Run()
}

// image step builds Docker images.
func image() error {
	var images = []struct {
		dir  string
		name string
	}{
		{"./bin/drone-agent", "drone/drone-agent"},
		{"./bin/drone-server", "drone/drone"},
	}
	for _, image := range images {
		path := filepath.Join(image.dir, "Dockerfile")
		name := image.name + ":" + version
		cmd := exec.Command("docker", "build", "-rm", path, name)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		trace(cmd.Args)
		err := cmd.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func clean() error {
	err := filepath.Walk(".", func(path string, f os.FileInfo, err error) error {
		suffixes := []string{
			".out",
			"_bindata.go",
		}

		for _, suffix := range suffixes {
			if strings.HasSuffix(path, suffix) {
				if err := os.Remove(path); err != nil {
					return err
				}
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	files := []string{
		"bin/drone",
		"bin/drone-agent",
		"bin/drone-build",
	}

	for _, file := range files {
		if _, err := os.Stat(file); err != nil {
			continue
		}

		if err := os.Remove(file); err != nil {
			return err
		}
	}

	return nil
}

// trace is a helper fucntion that writes a command
// to stdout similar to bash +x
func trace(args []string) {
	print("+ ")
	println(strings.Join(args, " "))
}

// helper function to parse the git revision
func rev() string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	raw, err := cmd.CombinedOutput()
	if err != nil {
		return "HEAD"
	}
	return strings.Trim(string(raw), "\n")
}
