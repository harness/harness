// +build ignore

// This program builds Drone.
// $ go run make.go build test
//
// The output binaries go into the ./bin/ directory (under the
// project root, where make.go is)
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	"build":   build,
	"test":    test,
	"image":   image,
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
	// concatinate scripts
	return nil
}

// styles step concatinates the css files.
func styles() error {
	// concatinate styles
	// inject css variables?
	return nil
}

// json step generates optimized json marshal and
// unmarshal functions to override defaults.
func json() error {
	return nil
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
