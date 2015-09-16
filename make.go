// +build ignore

// This program builds Drone.
// $ go run make.go deps bindata build test

package main

import (
	"fmt"
	"io/ioutil"
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
	"deps":    executeDeps,
	"json":    executeJson,
	"embed":   executeEmbed,
	"scripts": executeScripts,
	"styles":  executeStyles,
	"vet":     executeVet,
	"fmt":     executeFmt,
	"test":    executeTest,
	"build":   executeBuild,
	"install": executeInstall,
	"image":   executeImage,
	"bindata": executeBindata,
	"clean":   executeClean,
}

func main() {
	for _, arg := range os.Args[1:] {
		step, ok := steps[arg]

		if !ok {
			fmt.Println("Error: Invalid step", arg)
			os.Exit(1)
		}

		err := step()

		if err != nil {
			fmt.Println("Error: Failed step", arg)
			os.Exit(1)
		}
	}
}

type step func() error

func executeDeps() error {
	deps := []string{
		"github.com/jteeuwen/go-bindata/...",
		"golang.org/x/tools/cmd/cover",
	}

	for _, dep := range deps {
		err := run(
			"go",
			"get",
			"-u",
			dep)

		if err != nil {
			return err
		}
	}

	return nil
}

// json step generates optimized json marshal and
// unmarshal functions to override defaults.
func executeJson() error {
	return nil
}

// embed step embeds static files in .go files.
func executeEmbed() error {
	// embed drone.{revision}.css
	// embed drone.{revision}.js

	return nil
}

// scripts step concatinates all javascript files.
func executeScripts() error {
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
func executeStyles() error {
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

// vet step executes the `go vet` command
func executeVet() error {
	return run(
		"go",
		"vet",
		"github.com/drone/drone/pkg/...",
		"github.com/drone/drone/cmd/...")
}

// fmt step executes the `go fmt` command
func executeFmt() error {
	return run(
		"go",
		"fmt",
		"github.com/drone/drone/pkg/...",
		"github.com/drone/drone/cmd/...")
}

// test step executes unit tests and coverage.
func executeTest() error {
	ldf := fmt.Sprintf(
		"-X main.revision=%s -X main.version=%s",
		sha,
		version)

	return run(
		"go",
		"test",
		"-cover",
		"-ldflags",
		ldf,
		"github.com/drone/drone/pkg/...",
		"github.com/drone/drone/cmd/...")
}

// install step installs the application binaries.
func executeInstall() error {
	var bins = []struct {
		input string
	}{
		{
			"github.com/drone/drone/cmd/drone-server",
		},
	}

	for _, bin := range bins {
		ldf := fmt.Sprintf(
			"-X main.revision=%s -X main.version=%s",
			sha,
			version)

		err := run(
			"go",
			"install",
			"-ldflags",
			ldf,
			bin.input)

		if err != nil {
			return err
		}
	}

	return nil
}

// build step creates the application binaries.
func executeBuild() error {
	var bins = []struct {
		input  string
		output string
	}{
		{
			"github.com/drone/drone/cmd/drone-server",
			"bin/drone",
		},
	}

	for _, bin := range bins {
		ldf := fmt.Sprintf(
			"-X main.revision=%s -X main.version=%s",
			sha,
			version)

		err := run(
			"go",
			"build",
			"-o",
			bin.output,
			"-ldflags",
			ldf,
			bin.input)

		if err != nil {
			return err
		}
	}

	return nil
}

// image step builds docker images.
func executeImage() error {
	var images = []struct {
		dir  string
		name string
	}{
		{
			"bin/drone-server",
			"drone/drone",
		},
	}
	for _, image := range images {
		path := filepath.Join(
			image.dir,
			"Dockerfile")

		name := fmt.Sprintf("%s:%s",
			image.name,
			version)

		err := run(
			"docker",
			"build",
			"-rm",
			path,
			name)

		if err != nil {
			return err
		}
	}

	return nil
}

// bindata step generates go-bindata package.
func executeBindata() error {
	var paths = []struct {
		input  string
		output string
		pkg    string
	}{
		{
			"cmd/drone-server/static/...",
			"cmd/drone-server/drone_bindata.go",
			"main",
		},
	}

	for _, path := range paths {
		binErr := run(
			"go-bindata",
			fmt.Sprintf("-o=%s", path.output),
			fmt.Sprintf("-pkg=%s", path.pkg),
			path.input)

		if binErr != nil {
			return binErr
		}

		fmtErr := run(
			"go",
			"fmt",
			path.output)

		if fmtErr != nil {
			return fmtErr
		}
	}

	return nil
}

// clean step removes all generated files.
func executeClean() error {
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

// run is a helper function that executes commands
// and assigns stdout and stderr targets
func run(command string, args ...string) error {
	cmd := exec.Command(command, args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	trace(cmd.Args)
	return cmd.Run()
}

// helper function to parse the git revision
func rev() string {
	cmd := exec.Command(
		"git",
		"rev-parse",
		"--short",
		"HEAD")

	raw, err := cmd.CombinedOutput()

	if err != nil {
		return "HEAD"
	}

	return strings.Trim(string(raw), "\n")
}

// trace is a helper function that writes a command
// to stdout similar to bash +x
func trace(args []string) {
	print("+ ")
	println(strings.Join(args, " "))
}
