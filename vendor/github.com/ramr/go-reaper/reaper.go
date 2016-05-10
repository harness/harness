package reaper

/*  Note:  This is a *nix only implementation.  */

//  Prefer #include style directives.
import "fmt"
import "os"
import "os/signal"
import "syscall"


//  Handle death of child (SIGCHLD) messages. Pushes the signal onto the
//  notifications channel if there is a waiter.
func sigChildHandler(notifications chan os.Signal) {
	var sigs = make(chan os.Signal, 3)
	signal.Notify(sigs, syscall.SIGCHLD)

	for {
		var sig = <- sigs
		select {
		case notifications <-sig:  /*  published it.  */
		default:
			/*
			 *  Notifications channel full - drop it to the
			 *  floor. This ensures we don't fill up the SIGCHLD
			 *  queue. The reaper just waits for any child
			 *  process (pid=-1), so we ain't loosing it!! ;^)
			 */
		}
	}

}  /*  End of function  sigChildHandler.  */


//  Be a good parent - clean up behind the children.
func reapChildren() {
	var notifications = make(chan os.Signal, 1)

	go sigChildHandler(notifications)

	for {
		var sig = <-notifications
		fmt.Printf(" - Received signal %v\n", sig)
		for {
			var wstatus syscall.WaitStatus

			/*
			 *  Reap 'em, so that zombies don't accumulate.
			 *  Plants vs. Zombies!!
			 */
			pid, err := syscall.Wait4(-1, &wstatus, 0, nil)
			for syscall.EINTR == err {
				pid, err = syscall.Wait4(-1, &wstatus, 0, nil)
			}

			if syscall.ECHILD == err {
				break
			}

			fmt.Printf(" - Grim reaper cleanup: pid=%d, wstatus=%+v\n",
				pid, wstatus)

		}
	}

}  /*   End of function  reapChildren.  */



/*
 *  ======================================================================
 *  Section: Exported functions
 *  ======================================================================
 */

//  Entry point for the reaper code. Start reaping children in the
//  background inside a goroutine.
func Reap() {
	/*
	 *  Only reap processes if we are taking over init's duties aka
	 *  we are running as pid 1 inside a docker container.
	 */
	 if 1 == os.Getpid() {
		 /*
		  *  Ok, we are the grandma of 'em all, so we get to play
		  *  the grim reaper.
		  *  You will be missed, Terry Pratchett!! RIP
		  */
		  go reapChildren()
	 }

}  /*  End of [exported] function  Reap.  */

