package main

import "fmt"
import "os"
import "os/signal"
import "os/exec"
import "path/filepath"
import "syscall"

import reaper "github.com/ramr/go-reaper"


const NWORKERS=3


func start_workers() {
	//  Starts up workers - which in turn start up kids that get
	//  "orphaned".
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Printf(" - Error getting script dir - %s\n", err)
		return
	}

	var scriptFile = fmt.Sprintf("%s/bin/script.sh", dir)
	script, err := filepath.Abs(scriptFile)
	if err != nil {
		fmt.Printf(" - Error getting script - %s\n", scriptFile)
		return
	}
	
	var args = fmt.Sprintf("%d", NWORKERS)
	var cmd = exec.Command(script, args)
	cmd.Start()

	fmt.Printf("  - Started worker: %s %s\n", script, args)

}  /*  End of function  start_workers.  */



func main() {
  sig := make(chan os.Signal, 1)
  signal.Notify(sig, syscall.SIGUSR1)

  /*  Start the grim reaper ... */
  go reaper.Reap()

  /*  Start the initial set of workers ... */
  start_workers()

  for {
    select {
    case <-sig:
	fmt.Println("  - Got SIGUSR1, starting more workers ...")
	start_workers()
    }

  }  /*  End of while doomsday ... */

}  /*  End of function  main.  */
