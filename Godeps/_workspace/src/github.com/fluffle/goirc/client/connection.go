package client

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/fluffle/goevent/event"
	"github.com/fluffle/goirc/state"
	"github.com/fluffle/golog/logging"
	"net"
	"strings"
	"time"
)

// An IRC connection is represented by this struct.
type Conn struct {
	// Connection Hostname and Nickname
	Host     string
	Me       *state.Nick
	Network  string
	password string

	// Replaceable function to customise the 433 handler's new nick
	NewNick func(string) string

	// Event handler registry and dispatcher
	ER event.EventRegistry
	ED event.EventDispatcher

	// State tracker for nicks and channels
	ST state.StateTracker
	st bool

	// Use the State field to store external state that handlers might need.
	// Remember ... you might need locking for this ;-)
	State interface{}

	// I/O stuff to server
	sock      net.Conn
	io        *bufio.ReadWriter
	in        chan *Line
	out       chan string
	Connected bool

	// Control channels to goroutines
	cSend, cLoop, cPing chan bool

	// Misc knobs to tweak client behaviour:
	// Are we connecting via SSL? Do we care about certificate validity?
	SSL       bool
	SSLConfig *tls.Config

	// Client->server ping frequency, in seconds. Defaults to 3m.
	PingFreq time.Duration

	// Set this to true to disable flood protection and false to re-enable
	Flood bool

	// Internal counters for flood protection
	badness  time.Duration
	lastsent time.Time
}

// Creates a new IRC connection object, but doesn't connect to anything so
// that you can add event handlers to it. See AddHandler() for details.
func SimpleClient(nick string, args ...string) *Conn {
	r := event.NewRegistry()
	ident := "goirc"
	name := "Powered by GoIRC"

	if len(args) > 0 && args[0] != "" {
		ident = args[0]
	}
	if len(args) > 1 && args[1] != "" {
		name = args[1]
	}
	return Client(nick, ident, name, r)
}

func Client(nick, ident, name string, r event.EventRegistry) *Conn {
	if r == nil {
		return nil
	}
	logging.InitFromFlags()
	conn := &Conn{
		ER:        r,
		ED:        r,
		st:        false,
		in:        make(chan *Line, 32),
		out:       make(chan string, 32),
		cSend:     make(chan bool),
		cLoop:     make(chan bool),
		cPing:     make(chan bool),
		SSL:       false,
		SSLConfig: nil,
		PingFreq:  3 * time.Minute,
		Flood:     false,
		NewNick:   func(s string) string { return s + "_" },
		badness:   0,
		lastsent:  time.Now(),
	}
	conn.addIntHandlers()
	conn.Me = state.NewNick(nick)
	conn.Me.Ident = ident
	conn.Me.Name = name

	conn.initialise()
	return conn
}

func (conn *Conn) EnableStateTracking() {
	if !conn.st {
		n := conn.Me
		conn.ST = state.NewTracker(n.Nick)
		conn.Me = conn.ST.Me()
		conn.Me.Ident = n.Ident
		conn.Me.Name = n.Name
		conn.addSTHandlers()
		conn.st = true
	}
}

func (conn *Conn) DisableStateTracking() {
	if conn.st {
		conn.st = false
		conn.delSTHandlers()
		conn.ST.Wipe()
		conn.ST = nil
	}
}

// Per-connection state initialisation.
func (conn *Conn) initialise() {
	conn.io = nil
	conn.sock = nil
	if conn.st {
		conn.ST.Wipe()
	}
}

// Connect the IRC connection object to "host[:port]" which should be either
// a hostname or an IP address, with an optional port. To enable explicit SSL
// on the connection to the IRC server, set Conn.SSL to true before calling
// Connect(). The port will default to 6697 if ssl is enabled, and 6667
// otherwise. You can also provide an optional connect password.
func (conn *Conn) Connect(host string, pass ...string) error {
	if conn.Connected {
		return errors.New(fmt.Sprintf(
			"irc.Connect(): already connected to %s, cannot connect to %s",
			conn.Host, host))
	}

	if conn.SSL {
		if !hasPort(host) {
			host += ":6697"
		}
		logging.Info("irc.Connect(): Connecting to %s with SSL.", host)
		if s, err := tls.Dial("tcp", host, conn.SSLConfig); err == nil {
			conn.sock = s
		} else {
			return err
		}
	} else {
		if !hasPort(host) {
			host += ":6667"
		}
		logging.Info("irc.Connect(): Connecting to %s without SSL.", host)
		if s, err := net.Dial("tcp", host); err == nil {
			conn.sock = s
		} else {
			return err
		}
	}
	conn.Host = host
	if len(pass) > 0 {
		conn.password = pass[0]
	}
	conn.Connected = true
	conn.postConnect()
	conn.ED.Dispatch(INIT, conn, &Line{})
	return nil
}

// Post-connection setup (for ease of testing)
func (conn *Conn) postConnect() {
	conn.io = bufio.NewReadWriter(
		bufio.NewReader(conn.sock),
		bufio.NewWriter(conn.sock))
	go conn.send()
	go conn.recv()
	if conn.PingFreq > 0 {
		go conn.ping()
	} else {
		// Otherwise the send in shutdown will hang :-/
		go func() { <-conn.cPing }()
	}
	go conn.runLoop()
}

// copied from http.client for great justice
func hasPort(s string) bool {
	return strings.LastIndex(s, ":") > strings.LastIndex(s, "]")
}

// goroutine to pass data from output channel to write()
func (conn *Conn) send() {
	for {
		select {
		case line := <-conn.out:
			conn.write(line)
		case <-conn.cSend:
			// strobe on control channel, bail out
			return
		}
	}
}

// receive one \r\n terminated line from peer, parse and dispatch it
func (conn *Conn) recv() {
	for {
		s, err := conn.io.ReadString('\n')
		if err != nil {
			logging.Error("irc.recv(): %s", err.Error())
			conn.shutdown()
			return
		}
		s = strings.Trim(s, "\r\n")
		logging.Debug("<- %s", s)

		if line := parseLine(s); line != nil {
			line.Time = time.Now()
			conn.in <- line
		} else {
			logging.Warn("irc.recv(): problems parsing line:\n  %s", s)
		}
	}
}

// Repeatedly pings the server every PingFreq seconds (no matter what)
func (conn *Conn) ping() {
	tick := time.NewTicker(conn.PingFreq)
	for {
		select {
		case <-tick.C:
			conn.Raw(fmt.Sprintf("PING :%d", time.Now().UnixNano()))
		case <-conn.cPing:
			tick.Stop()
			return
		}
	}
}

// goroutine to dispatch events for lines received on input channel
func (conn *Conn) runLoop() {
	for {
		select {
		case line := <-conn.in:
			conn.ED.Dispatch(line.Cmd, conn, line)
		case <-conn.cLoop:
			// strobe on control channel, bail out
			return
		}
	}
}

// Write a \r\n terminated line of output to the connected server,
// using Hybrid's algorithm to rate limit if conn.Flood is false.
func (conn *Conn) write(line string) {
	if !conn.Flood {
		if t := conn.rateLimit(len(line)); t != 0 {
			// sleep for the current line's time value before sending it
			logging.Debug("irc.rateLimit(): Flood! Sleeping for %.2f secs.",
				t.Seconds())
			<-time.After(t)
		}
	}

	if _, err := conn.io.WriteString(line + "\r\n"); err != nil {
		logging.Error("irc.send(): %s", err.Error())
		conn.shutdown()
		return
	}
	if err := conn.io.Flush(); err != nil {
		logging.Error("irc.send(): %s", err.Error())
		conn.shutdown()
		return
	}
	logging.Debug("-> %s", line)
}

// Implement Hybrid's flood control algorithm to rate-limit outgoing lines.
func (conn *Conn) rateLimit(chars int) time.Duration {
	// Hybrid's algorithm allows for 2 seconds per line and an additional
	// 1/120 of a second per character on that line.
	linetime := 2*time.Second + time.Duration(chars)*time.Second/120
	elapsed := time.Now().Sub(conn.lastsent)
	if conn.badness += linetime - elapsed; conn.badness < 0 {
		// negative badness times are badness...
		conn.badness = 0
	}
	conn.lastsent = time.Now()
	// If we've sent more than 10 second's worth of lines according to the
	// calculation above, then we're at risk of "Excess Flood".
	if conn.badness > 10*time.Second {
		return linetime
	}
	return 0
}

func (conn *Conn) shutdown() {
	// Guard against double-call of shutdown() if we get an error in send()
	// as calling sock.Close() will cause recv() to recieve EOF in readstring()
	if conn.Connected {
		logging.Info("irc.shutdown(): Disconnected from server.")
		conn.ED.Dispatch(DISCONNECTED, conn, &Line{})
		conn.Connected = false
		conn.sock.Close()
		conn.cSend <- true
		conn.cLoop <- true
		conn.cPing <- true
		// reinit datastructures ready for next connection
		// do this here rather than after runLoop()'s for due to race
		conn.initialise()
	}
}

// Dumps a load of information about the current state of the connection to a
// string for debugging state tracking and other such things.
func (conn *Conn) String() string {
	str := "GoIRC Connection\n"
	str += "----------------\n\n"
	if conn.Connected {
		str += "Connected to " + conn.Host + "\n\n"
	} else {
		str += "Not currently connected!\n\n"
	}
	str += conn.Me.String() + "\n"
	if conn.st {
		str += conn.ST.String() + "\n"
	}
	return str
}
