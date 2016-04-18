package watch

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/smartystreets/goconvey/web/server/messaging"
)

func TestWatcher(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping potentially long-running integration test...")
		return
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
	output := new(bytes.Buffer)
	log.SetOutput(output)
	defer func() { t.Log(output.String()) }()

	_, filename, _, _ := runtime.Caller(0)
	originalRoot := filepath.Join(filepath.Dir(filename), "integration_testing")
	temporary, err := ioutil.TempDir("/tmp", "goconvey")
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(temporary, "integration_testing")
	sub := filepath.Join(root, "sub")

	err = CopyDir(originalRoot, root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err = os.RemoveAll(temporary)
		if err != nil {
			t.Fatal(err)
		}
	}()

	var ( // commands
		pause  = messaging.WatcherCommand{Instruction: messaging.WatcherPause}
		resume = messaging.WatcherCommand{Instruction: messaging.WatcherResume}

		ignore    = messaging.WatcherCommand{Instruction: messaging.WatcherIgnore, Details: sub}
		reinstate = messaging.WatcherCommand{Instruction: messaging.WatcherReinstate, Details: sub}

		adjustToSub  = messaging.WatcherCommand{Instruction: messaging.WatcherAdjustRoot, Details: sub}
		adjustToRoot = messaging.WatcherCommand{Instruction: messaging.WatcherAdjustRoot, Details: root}

		execute = messaging.WatcherCommand{Instruction: messaging.WatcherExecute}

		bogus = messaging.WatcherCommand{Instruction: 42}

		stop = messaging.WatcherCommand{Instruction: messaging.WatcherStop}
	)

	Convey("Subject: Watcher operations", t, func() {
		input := make(chan messaging.WatcherCommand)
		output := make(chan messaging.Folders)
		excludedDirs := []string{}
		watcher := NewWatcher(root, -1, time.Millisecond, input, output, ".go", excludedDirs)

		go watcher.Listen()

		Convey("Initial scan results", func() {
			go func() { input <- stop }()

			results := []messaging.Folders{}
			for result := range output {
				results = append(results, result)
			}

			So(len(results), ShouldEqual, 1)
		})

		Convey("Manual execution produces additional results", func() {
			go func() {
				input <- execute
				input <- stop
			}()

			results := []messaging.Folders{}
			for result := range output {
				results = append(results, result)
			}

			So(len(results), ShouldEqual, 2)
			So(len(results[0]), ShouldEqual, 2)
			So(len(results[1]), ShouldEqual, 2)
		})

		Convey("Ignore and Reinstate commands are not reflected in the scan results", func() {
			go func() {
				input <- ignore
				input <- reinstate
				input <- stop
			}()

			results := []messaging.Folders{}
			for result := range output {
				results = append(results, result)
			}

			So(len(results), ShouldEqual, 1)
			So(results[0][sub].Ignored, ShouldBeFalse) // initial
		})

		Convey("Adjusting the root changes the number of folders in the scanned results", func() {
			go func() {
				input <- adjustToSub
				input <- adjustToRoot
				input <- stop
			}()

			results := []messaging.Folders{}
			for result := range output {
				results = append(results, result)
			}

			So(len(results), ShouldEqual, 3)
			So(len(results[0]), ShouldEqual, 2) // initial
			So(len(results[1]), ShouldEqual, 1) // root moved to sub
			So(len(results[2]), ShouldEqual, 2) // root moved back to original location
		})

		Convey("A bogus command does not cause any additional scanning beyond the initial scan", func() {
			go func() {
				input <- bogus
				input <- stop
			}()

			results := []messaging.Folders{}
			for result := range output {
				results = append(results, result)
			}

			So(len(results), ShouldEqual, 1) // initial scan
		})

		Convey("Scanning occurs as a result of a file system update", func() {
			go func() {
				time.Sleep(time.Second)
				exec.Command("touch", filepath.Join(root, "main.go")).Run()
				time.Sleep(time.Second)
				input <- stop
			}()

			results := []messaging.Folders{}
			for result := range output {
				results = append(results, result)
			}

			So(len(results), ShouldEqual, 2)
		})

		Convey("Scanning does not occur as a result of resuming after a pause", func() {
			go func() {
				input <- pause
				input <- resume
				input <- stop
			}()

			results := []messaging.Folders{}
			for result := range output {
				results = append(results, result)
			}

			So(len(results), ShouldEqual, 1)
		})

		Convey("Scanning should not occur when the watcher is paused", func() {
			go func() {
				input <- pause
				for x := 0; x < 2; x++ {
					time.Sleep(time.Millisecond * 250)
					exec.Command("touch", filepath.Join(root, "main.go")).Run()
					time.Sleep(time.Millisecond * 250)
				}
				input <- resume
				input <- stop
			}()

			results := []messaging.Folders{}
			for result := range output {
				results = append(results, result)
			}

			So(len(results), ShouldEqual, 2)
		})
	})
}
