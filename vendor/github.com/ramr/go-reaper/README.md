# go-reaper
Process (grim) reaper library for golang - this is useful for cleaning up
zombie processes inside docker containers (which do not have an init
process running as pid 1).


tl;dr
-----

       import reaper "github.com/ramr/go-reaper"

       func main() {
		//  Start background reaping of orphaned child processes.
		go reaper.Reap()

		//  Rest of your code ...
       }



How and Why
-----------
If you run a container without an init process (pid 1) which would
normally reap zombie processes, you could well end up with a lot of zombie
processes and eventually exhaust the max process limit on your system.

If you have a golang program that runs as pid 1, then this library allows
the golang program to setup a background signal handling mechanism to
handle the death of those orphaned children and not create a load of
zombies inside the pid namespace your container runs in.


Usage:
------
See the tl;dr section above.

