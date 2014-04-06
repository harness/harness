package client

import (
	"bufio"
	"code.google.com/p/gomock/gomock"
	"github.com/fluffle/goevent/event"
	"github.com/fluffle/golog/logging"
	"github.com/fluffle/goirc/state"
	"strings"
	"testing"
	"time"
)

type testState struct {
	ctrl *gomock.Controller
	st   *state.MockStateTracker
	ed   *event.MockEventDispatcher
	nc   *mockNetConn
	c    *Conn
}

func setUp(t *testing.T, start ...bool) (*Conn, *testState) {
	ctrl := gomock.NewController(t)
	st := state.NewMockStateTracker(ctrl)
	r := event.NewRegistry()
	ed := event.NewMockEventDispatcher(ctrl)
	nc := MockNetConn(t)
	c := Client("test", "test", "Testing IRC", r)
	logging.SetLogLevel(logging.LogFatal)

	c.ED = ed
	c.ST = st
	c.st = true
	c.sock = nc
	c.Flood = true // Tests can take a while otherwise
	c.Connected = true
	if len(start) == 0 {
		// Hack to allow tests of send, recv, write etc.
		// NOTE: the value of the boolean doesn't matter.
		c.postConnect()
		// Sleep 1ms to allow background routines to start.
		<-time.After(1e6)
	}

	return c, &testState{ctrl, st, ed, nc, c}
}

func (s *testState) tearDown() {
	s.ed.EXPECT().Dispatch("disconnected", s.c, &Line{})
	s.st.EXPECT().Wipe()
	s.nc.ExpectNothing()
	s.c.shutdown()
	<-time.After(1e6)
	s.ctrl.Finish()
}

// Practically the same as the above test, but shutdown is called implicitly
// by recv() getting an EOF from the mock connection.
func TestEOF(t *testing.T) {
	c, s := setUp(t)
	// Since we're not using tearDown() here, manually call Finish()
	defer s.ctrl.Finish()

	// Simulate EOF from server
	s.ed.EXPECT().Dispatch("disconnected", c, &Line{})
	s.st.EXPECT().Wipe()
	s.nc.Close()

	// Since things happen in different internal goroutines, we need to wait
	// 1 ms should be enough :-)
	<-time.After(1e6)

	// Verify that the connection no longer thinks it's connected
	if c.Connected {
		t.Errorf("Conn still thinks it's connected to the server.")
	}
}

func TestClientAndStateTracking(t *testing.T) {
	// This doesn't use setUp() as we want to pass in a mock EventRegistry.
	ctrl := gomock.NewController(t)
	r := event.NewMockEventRegistry(ctrl)
	st := state.NewMockStateTracker(ctrl)

	for n, _ := range intHandlers {
		// We can't use EXPECT() here as comparisons of functions are
		// no longer valid in Go, which causes reflect.DeepEqual to bail.
		// Instead, ignore the function arg and just ensure that all the
		// handler names are correctly passed to AddHandler.
		ctrl.RecordCall(r, "AddHandler", gomock.Any(), n)
	}
	c := Client("test", "test", "Testing IRC", r)

	// Assert some basic things about the initial state of the Conn struct
	if c.ER != r || c.ED != r || c.st != false || c.ST != nil {
		t.Errorf("Conn not correctly initialised with external deps.")
	}
	if c.in == nil || c.out == nil || c.cSend == nil || c.cLoop == nil {
		t.Errorf("Conn control channels not correctly initialised.")
	}
	if c.Me.Nick != "test" || c.Me.Ident != "test" ||
		c.Me.Name != "Testing IRC" || c.Me.Host != "" {
		t.Errorf("Conn.Me not correctly initialised.")
	}

	// OK, while we're here with a mock event registry...
	for n, _ := range stHandlers {
		// See above.
		ctrl.RecordCall(r, "AddHandler", gomock.Any(), n)
	}
	c.EnableStateTracking()

	// We're expecting the untracked me to be replaced by a tracked one.
	if c.Me.Nick != "test" || c.Me.Ident != "test" ||
		c.Me.Name != "Testing IRC" || c.Me.Host != "" {
		t.Errorf("Enabling state tracking did not replace Me correctly.")
	}
	if !c.st || c.ST == nil || c.Me != c.ST.Me() {
		t.Errorf("State tracker not enabled correctly.")
	}

	// Now, shim in the mock state tracker and test disabling state tracking.
	me := c.Me
	c.ST = st
	st.EXPECT().Wipe()
	for n, _ := range stHandlers {
		// See above.
		ctrl.RecordCall(r, "DelHandler", gomock.Any(), n)
	}
	c.DisableStateTracking()
	if c.st || c.ST != nil || c.Me != me {
		t.Errorf("State tracker not disabled correctly.")
	}
	ctrl.Finish()
}

func TestSend(t *testing.T) {
	// Passing a second value to setUp inhibits postConnect()
	c, s := setUp(t, false)
	// We can't use tearDown here, as it will cause a deadlock in shutdown()
	// trying to send kill messages down channels to nonexistent goroutines.
	defer s.ctrl.Finish()

	// ... so we have to do some of it's work here.
	c.io = bufio.NewReadWriter(
		bufio.NewReader(c.sock),
		bufio.NewWriter(c.sock))

	// Assert that before send is running, nothing should be sent to the socket
	// but writes to the buffered channel "out" should not block.
	c.out <- "SENT BEFORE START"
	s.nc.ExpectNothing()

	// We want to test that the a goroutine calling send will exit correctly.
	exited := false
	go func() {
		c.send()
		exited = true
	}()

	// send is now running in the background as if started by postConnect.
	// This should read the line previously buffered in c.out, and write it
	// to the socket connection.
	s.nc.Expect("SENT BEFORE START")

	// Send another line, just to be sure :-)
	c.out <- "SENT AFTER START"
	s.nc.Expect("SENT AFTER START")

	// Now, use the control channel to exit send and kill the goroutine.
	if exited {
		t.Errorf("Exited before signal sent.")
	}
	c.cSend <- true
	// Allow propagation time...
	<-time.After(1e6)
	if !exited {
		t.Errorf("Didn't exit after signal.")
	}
	s.nc.ExpectNothing()

	// Sending more on c.out shouldn't reach the network.
	c.out <- "SENT AFTER END"
	s.nc.ExpectNothing()
}

func TestRecv(t *testing.T) {
	// Passing a second value to setUp inhibits postConnect()
	c, s := setUp(t, false)
	// We can't tearDown here as we need to explicitly test recv exiting.
	// The same shutdown() caveat in TestSend above also applies.
	defer s.ctrl.Finish()

	// ... so we have to do some of it's work here.
	c.io = bufio.NewReadWriter(
		bufio.NewReader(c.sock),
		bufio.NewWriter(c.sock))

	// Send a line before recv is started up, to verify nothing appears on c.in
	s.nc.Send(":irc.server.org 001 test :First test line.")

	// reader is a helper to do a "non-blocking" read of c.in
	reader := func() *Line {
		select {
		case <-time.After(1e6):
		case l := <-c.in:
			return l
		}
		return nil
	}
	if l := reader(); l != nil {
		t.Errorf("Line parsed before recv started.")
	}

	// We want to test that the a goroutine calling recv will exit correctly.
	exited := false
	go func() {
		c.recv()
		exited = true
	}()

	// Strangely, recv() needs some time to start up, but *only* when this test
	// is run standalone with: client/_test/_testmain --test.run TestRecv
	<-time.After(1e6)

	// Now, this should mean that we'll receive our parsed line on c.in
	if l := reader(); l == nil || l.Cmd != "001" {
		t.Errorf("Bad first line received on input channel")
	}

	// Send a second line, just to be sure.
	s.nc.Send(":irc.server.org 002 test :Second test line.")
	if l := reader(); l == nil || l.Cmd != "002" {
		t.Errorf("Bad second line received on input channel.")
	}

	// Test that recv does something useful with a line it can't parse
	// (not that there are many, parseLine is forgiving).
	s.nc.Send(":textwithnospaces")
	if l := reader(); l != nil {
		t.Errorf("Bad line still caused receive on input channel.")
	}

	// The only way recv() exits is when the socket closes.
	if exited {
		t.Errorf("Exited before socket close.")
	}
	s.ed.EXPECT().Dispatch("disconnected", c, &Line{})
	s.st.EXPECT().Wipe()
	s.nc.Close()

	// Since send and runloop aren't actually running, we need to empty their
	// channels manually for recv() to be able to call shutdown correctly.
	<-c.cSend
	<-c.cLoop
	<-c.cPing
	// Give things time to shake themselves out...
	<-time.After(1e6)
	if !exited {
		t.Errorf("Didn't exit on socket close.")
	}

	// Since s.nc is closed we can't attempt another send on it...
	if l := reader(); l != nil {
		t.Errorf("Line received on input channel after socket close.")
	}
}

func TestPing(t *testing.T) {
	// Passing a second value to setUp inhibits postConnect()
	c, s := setUp(t, false)
	// We can't use tearDown here, as it will cause a deadlock in shutdown()
	// trying to send kill messages down channels to nonexistent goroutines.
	defer s.ctrl.Finish()

	// Set a low ping frequency for testing.
	c.PingFreq = 50 * time.Millisecond

	// reader is a helper to do a "non-blocking" read of c.out
	reader := func() string {
		select {
		case <-time.After(time.Millisecond):
		case s := <-c.out:
			return s
		}
		return ""
	}
	if s := reader(); s != "" {
		t.Errorf("Line output before ping started.")
	}

	// Start ping loop.
	exited := false
	go func() {
		c.ping()
		exited = true
	}()

	// The first ping should be after a second,
	// so we don't expect anything now on c.in
	if s := reader(); s != "" {
		t.Errorf("Line output directly after ping started.")
	}

	<-time.After(50 * time.Millisecond)
	if s := reader(); s == "" || !strings.HasPrefix(s, "PING :") {
		t.Errorf("Line not output after 50ms.")
	}

	// Reader waits for 1ms and we call it a few times above.
	<-time.After(45 * time.Millisecond)
	if s := reader(); s != "" {
		t.Errorf("Line output under 50ms after last ping.")
	}

	// This is a short window (49-51ms) in which the ping should happen
	// This may result in flaky tests; sorry (and file a bug) if so.
	<-time.After(2 * time.Millisecond)
	if s := reader(); s == "" || !strings.HasPrefix(s, "PING :") {
		t.Errorf("Line not output after another 2ms.")
	}

	// Now kill the ping loop.
	if exited {
		t.Errorf("Exited before signal sent.")
	}

	c.cPing <- true
	// Make sure we're no longer pinging by waiting ~2x PingFreq
	<-time.After(105 * time.Millisecond)
	if s := reader(); s != "" {
		t.Errorf("Line output after ping stopped.")
	}

	if !exited {
		t.Errorf("Didn't exit after signal.")
	}
}

func TestRunLoop(t *testing.T) {
	// Passing a second value to setUp inhibits postConnect()
	c, s := setUp(t, false)
	// We can't use tearDown here, as it will cause a deadlock in shutdown()
	// trying to send kill messages down channels to nonexistent goroutines.
	defer s.ctrl.Finish()

	// ... so we have to do some of it's work here.
	c.io = bufio.NewReadWriter(
		bufio.NewReader(c.sock),
		bufio.NewWriter(c.sock))

	// NOTE: here we assert that no Dispatch event has been called yet by
	// calling s.ctrl.Finish(). There doesn't appear to be any harm in this.
	l1 := parseLine(":irc.server.org 001 test :First test line.")
	c.in <- l1
	s.ctrl.Finish()

	// We want to test that the a goroutine calling runLoop will exit correctly.
	// Now, we can expect the call to Dispatch to take place as runLoop starts.
	s.ed.EXPECT().Dispatch("001", c, l1)
	exited := false
	go func() {
		c.runLoop()
		exited = true
	}()
	// Here, the opposite seemed to take place, with TestRunLoop failing when
	// run as part of the suite but passing when run on it's own.
	<-time.After(1e6)

	// Send another line, just to be sure :-)
	l2 := parseLine(":irc.server.org 002 test :Second test line.")
	s.ed.EXPECT().Dispatch("002", c, l2)
	c.in <- l2
	// It appears some sleeping is needed after all of these to ensure channel
	// sends occur before the close signal is sent below...
	<-time.After(1e6)

	// Now, use the control channel to exit send and kill the goroutine.
	if exited {
		t.Errorf("Exited before signal sent.")
	}
	c.cLoop <- true
	// Allow propagation time...
	<-time.After(1e6)
	if !exited {
		t.Errorf("Didn't exit after signal.")
	}

	// Sending more on c.in shouldn't dispatch any further events
	c.in <- l1
}

func TestWrite(t *testing.T) {
	// Passing a second value to setUp inhibits postConnect()
	c, s := setUp(t, false)
	// We can't use tearDown here, as it will cause a deadlock in shutdown()
	// trying to send kill messages down channels to nonexistent goroutines.
	defer s.ctrl.Finish()

	// ... so we have to do some of it's work here.
	c.io = bufio.NewReadWriter(
		bufio.NewReader(c.sock),
		bufio.NewWriter(c.sock))

	// Write should just write a line to the socket.
	c.write("yo momma")
	s.nc.Expect("yo momma")

	// Flood control is disabled -- setUp sets c.Flood = true -- so we should
	// not have set c.badness at this point.
	if c.badness != 0 {
		t.Errorf("Flood control used when Flood = true.")
	}

	c.Flood = false
	c.write("she so useless")
	s.nc.Expect("she so useless")

	// The lastsent time should have been updated very recently...
	if time.Now().Sub(c.lastsent) > time.Millisecond {
		t.Errorf("Flood control not used when Flood = false.")
	}

	// Finally, test the error state by closing the socket then writing.
	// This little function makes sure that all the blocking channels that are
	// written to during the course of s.nc.Close() and c.write() are read from
	// again, to prevent deadlocks when these are both called synchronously.
	// XXX: This may well be a horrible hack.
	go func() {
		<-c.cSend
		<-c.cLoop
		<-c.cPing
	}()
	s.nc.Close()

	s.ed.EXPECT().Dispatch("disconnected", c, &Line{})
	s.st.EXPECT().Wipe()
	c.write("she can't pass unit tests")
}

func TestRateLimit(t *testing.T) {
	c, s := setUp(t)
	defer s.tearDown()

	if c.badness != 0 {
		t.Errorf("Bad initial values for rate limit variables.")
	}

	// We'll be needing this later...
	abs := func(i time.Duration) time.Duration {
		if (i < 0) {
			return -i
		}
		return i
	}

	// Since the changes to the time module, c.lastsent is now a time.Time.
	// It's initialised on client creation to time.Now() which for the purposes
	// of this test was probably around 1.2 ms ago. This is inconvenient.
	// Making it >10s ago effectively clears out the inconsistency, as this
	// makes elapsed > linetime and thus zeros c.badness and resets c.lastsent.
	c.lastsent = time.Now().Add(-10 * time.Second)
	if l := c.rateLimit(60); l != 0 || c.badness != 0 {
		t.Errorf("Rate limit got non-zero badness from long-ago lastsent.")
	}

	// So, time at the nanosecond resolution is a bit of a bitch. Choosing 60
	// characters as the line length means we should be increasing badness by
	// 2.5 seconds minus the delta between the two ratelimit calls. This should
	// be minimal but it's guaranteed that it won't be zero. Use 10us as a fuzz.
	if l := c.rateLimit(60); l != 0 || abs(c.badness - 25*1e8) > 10 * time.Microsecond {
		t.Errorf("Rate limit calculating badness incorrectly.")
	}
	// At this point, we can tip over the badness scale, with a bit of help.
	// 720 chars => +8 seconds of badness => 10.5 seconds => ratelimit
	if l := c.rateLimit(720); l != 8 * time.Second ||
		abs(c.badness - 105*1e8) > 10 * time.Microsecond {
		t.Errorf("Rate limit failed to return correct limiting values.")
		t.Errorf("l=%d, badness=%d", l, c.badness)
	}
}
