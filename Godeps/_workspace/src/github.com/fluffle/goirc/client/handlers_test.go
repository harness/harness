package client

import (
	"code.google.com/p/gomock/gomock"
	"github.com/fluffle/goirc/state"
	"testing"
)

// This test performs a simple end-to-end verification of correct line parsing
// and event dispatch as well as testing the PING handler. All the other tests
// in this file will call their respective handlers synchronously, otherwise
// testing becomes more difficult.
func TestPING(t *testing.T) {
	c, s := setUp(t)
	defer s.tearDown()
	// As this is a real end-to-end test, we need a real end-to-end dispatcher.
	c.ED = c.ER
	s.nc.Send("PING :1234567890")
	s.nc.Expect("PONG :1234567890")
	// Return mock dispatcher to it's rightful place afterwards for tearDown.
	c.ED = s.ed
}

// Test that the inbuilt INIT handler does the right things
func TestINIT(t *testing.T) {
	c, s := setUp(t)
	defer s.tearDown()

	c.h_INIT(&Line{})
	s.nc.Expect("NICK test")
	s.nc.Expect("USER test 12 * :Testing IRC")
	s.nc.ExpectNothing()

	c.password = "12345"
	c.Me.Ident = "idiot"
	c.Me.Name = "I've got the same combination on my luggage!"
	c.h_INIT(&Line{})
	s.nc.Expect("PASS 12345")
	s.nc.Expect("NICK test")
	s.nc.Expect("USER idiot 12 * :I've got the same combination on my luggage!")
	s.nc.ExpectNothing()
}

// Test the handler for 001 / RPL_WELCOME
func Test001(t *testing.T) {
	c, s := setUp(t)
	defer s.tearDown()

	l := parseLine(":irc.server.org 001 test :Welcome to IRC test!ident@somehost.com")
	s.ed.EXPECT().Dispatch("connected", c, l)
	// Call handler with a valid 001 line
	c.h_001(l)

	// Check host parsed correctly
	if c.Me.Host != "somehost.com" {
		t.Errorf("Host parsing failed, host is '%s'.", c.Me.Host)
	}
}

// Test the handler for 433 / ERR_NICKNAMEINUSE
func Test433(t *testing.T) {
	c, s := setUp(t)
	defer s.tearDown()

	// Call handler with a 433 line, not triggering c.Me.Renick()
	c.h_433(parseLine(":irc.server.org 433 test new :Nickname is already in use."))
	s.nc.Expect("NICK new_")

	// In this case, we're expecting the server to send a NICK line
	if c.Me.Nick != "test" {
		t.Errorf("ReNick() called unexpectedly, Nick == '%s'.", c.Me.Nick)
	}

	// Send a line that will trigger a renick. This happens when our wanted
	// nick is unavailable during initial negotiation, so we must choose a
	// different one before the connection can proceed. No NICK line will be
	// sent by the server to confirm nick change in this case.
	s.st.EXPECT().ReNick("test", "test_")
	c.h_433(parseLine(":irc.server.org 433 test test :Nickname is already in use."))
	s.nc.Expect("NICK test_")

	// Counter-intuitively, c.Me.Nick will not change in this case. This is an
	// artifact of the test set-up, with a mocked out state tracker that
	// doesn't actually change any state. Normally, this would be fine :-)
	if c.Me.Nick != "test" {
		t.Errorf("My nick changed from '%s'.", c.Me.Nick)
	}

	// Test the code path that *doesn't* involve state tracking.
	c.st = false
	c.h_433(parseLine(":irc.server.org 433 test test :Nickname is already in use."))
	s.nc.Expect("NICK test_")

	if c.Me.Nick != "test_" {
		t.Errorf("My nick not updated from '%s'.", c.Me.Nick)
	}
	c.st = true
}

// Test the handler for NICK messages when state tracking is disabled
func TestNICK(t *testing.T) {
	c, s := setUp(t)
	defer s.tearDown()

	// State tracking is enabled by default in setUp
	c.st = false

	// Call handler with a NICK line changing "our" nick to test1.
	c.h_NICK(parseLine(":test!test@somehost.com NICK :test1"))

	// Verify that our Nick has changed
	if c.Me.Nick != "test1" {
		t.Errorf("NICK did not result in changing our nick.")
	}

	// Send a NICK line for something that isn't us.
	c.h_NICK(parseLine(":blah!moo@cows.com NICK :milk"))

	// Verify that our Nick hasn't changed
	if c.Me.Nick != "test1" {
		t.Errorf("NICK did not result in changing our nick.")
	}

	// Re-enable state tracking and send a line that *should* change nick.
	c.st = true
	c.h_NICK(parseLine(":test1!test@somehost.com NICK :test2"))

	// Verify that our Nick hasn't changed (should be handled by h_STNICK).
	if c.Me.Nick != "test1" {
		t.Errorf("NICK changed our nick when state tracking enabled.")
	}
}

// Test the handler for CTCP messages
func TestCTCP(t *testing.T) {
	c, s := setUp(t)
	defer s.tearDown()

	// Call handler with CTCP VERSION
	c.h_CTCP(parseLine(":blah!moo@cows.com PRIVMSG test :\001VERSION\001"))

	// Expect a version reply
	s.nc.Expect("NOTICE blah :\001VERSION powered by goirc...\001")

	// Call handler with CTCP PING
	c.h_CTCP(parseLine(":blah!moo@cows.com PRIVMSG test :\001PING 1234567890\001"))

	// Expect a ping reply
	s.nc.Expect("NOTICE blah :\001PING 1234567890\001")

	// Call handler with CTCP UNKNOWN
	c.h_CTCP(parseLine(":blah!moo@cows.com PRIVMSG test :\001UNKNOWN ctcp\001"))
}

// Test the handler for JOIN messages
func TestJOIN(t *testing.T) {
	c, s := setUp(t)
	defer s.tearDown()

	// The state tracker should be creating a new channel in this first test
	chan1 := state.NewChannel("#test1")

	gomock.InOrder(
		s.st.EXPECT().GetChannel("#test1").Return(nil),
		s.st.EXPECT().GetNick("test").Return(c.Me),
		s.st.EXPECT().NewChannel("#test1").Return(chan1),
		s.st.EXPECT().Associate(chan1, c.Me),
	)

	// Use #test1 to test expected behaviour
	// Call handler with JOIN by test to #test1
	c.h_JOIN(parseLine(":test!test@somehost.com JOIN :#test1"))

	// Verify that the MODE and WHO commands are sent correctly
	s.nc.Expect("MODE #test1")
	s.nc.Expect("WHO #test1")

	// In this second test, we should be creating a new nick
	nick1 := state.NewNick("user1")

	gomock.InOrder(
		s.st.EXPECT().GetChannel("#test1").Return(chan1),
		s.st.EXPECT().GetNick("user1").Return(nil),
		s.st.EXPECT().NewNick("user1").Return(nick1),
		s.st.EXPECT().Associate(chan1, nick1),
	)

	// OK, now #test1 exists, JOIN another user we don't know about
	c.h_JOIN(parseLine(":user1!ident1@host1.com JOIN :#test1"))

	// Verify that the WHO command is sent correctly
	s.nc.Expect("WHO user1")

	// In this third test, we'll be pretending we know about the nick already.
	nick2 := state.NewNick("user2")
	gomock.InOrder(
		s.st.EXPECT().GetChannel("#test1").Return(chan1),
		s.st.EXPECT().GetNick("user2").Return(nick2),
		s.st.EXPECT().Associate(chan1, nick2),
	)
	c.h_JOIN(parseLine(":user2!ident2@host2.com JOIN :#test1"))

	// Test error paths
	gomock.InOrder(
		// unknown channel, unknown nick
		s.st.EXPECT().GetChannel("#test2").Return(nil),
		s.st.EXPECT().GetNick("blah").Return(nil),
		// unknown channel, known nick that isn't Me.
		s.st.EXPECT().GetChannel("#test2").Return(nil),
		s.st.EXPECT().GetNick("user2").Return(nick2),
	)
	c.h_JOIN(parseLine(":blah!moo@cows.com JOIN :#test2"))
	c.h_JOIN(parseLine(":user2!ident2@host2.com JOIN :#test2"))
}

// Test the handler for PART messages
func TestPART(t *testing.T) {
	c, s := setUp(t)
	defer s.tearDown()

	// We need some valid and associated nicks / channels to PART with.
	chan1 := state.NewChannel("#test1")
	nick1 := state.NewNick("user1")

	// PART should dissociate a nick from a channel.
	gomock.InOrder(
		s.st.EXPECT().GetChannel("#test1").Return(chan1),
		s.st.EXPECT().GetNick("user1").Return(nick1),
		s.st.EXPECT().Dissociate(chan1, nick1),
	)
	c.h_PART(parseLine(":user1!ident1@host1.com PART #test1 :Bye!"))
}

// Test the handler for KICK messages
// (this is very similar to the PART message test)
func TestKICK(t *testing.T) {
	c, s := setUp(t)
	defer s.tearDown()

	// We need some valid and associated nicks / channels to KICK.
	chan1 := state.NewChannel("#test1")
	nick1 := state.NewNick("user1")

	// KICK should dissociate a nick from a channel.
	gomock.InOrder(
		s.st.EXPECT().GetChannel("#test1").Return(chan1),
		s.st.EXPECT().GetNick("user1").Return(nick1),
		s.st.EXPECT().Dissociate(chan1, nick1),
	)
	c.h_KICK(parseLine(":test!test@somehost.com KICK #test1 user1 :Bye!"))
}

// Test the handler for QUIT messages
func TestQUIT(t *testing.T) {
	c, s := setUp(t)
	defer s.tearDown()

	// Have user1 QUIT. All possible errors handled by state tracker \o/
	s.st.EXPECT().DelNick("user1")
	c.h_QUIT(parseLine(":user1!ident1@host1.com QUIT :Bye!"))
}

// Test the handler for MODE messages
func TestMODE(t *testing.T) {
	c, s := setUp(t)
	defer s.tearDown()

	chan1 := state.NewChannel("#test1")
	nick1 := state.NewNick("user1")

	// Send a channel mode line. Inconveniently, Channel and Nick objects
	// aren't mockable with gomock as they're not interface types (and I
	// don't want them to be, writing accessors for struct fields sucks).
	// This makes testing whether ParseModes is called correctly harder.
	s.st.EXPECT().GetChannel("#test1").Return(chan1)
	c.h_MODE(parseLine(":user1!ident1@host1.com MODE #test1 +sk somekey"))
	if !chan1.Modes.Secret || chan1.Modes.Key != "somekey" {
		t.Errorf("Channel.ParseModes() not called correctly.")
	}


	// Send a nick mode line, returning Me
	gomock.InOrder(
		s.st.EXPECT().GetChannel("test").Return(nil),
		s.st.EXPECT().GetNick("test").Return(c.Me),
	)
	c.h_MODE(parseLine(":test!test@somehost.com MODE test +i"))
	if !c.Me.Modes.Invisible {
		t.Errorf("Nick.ParseModes() not called correctly.")
	}

	// Check error paths
	gomock.InOrder(
		// send a valid user mode that's not us
		s.st.EXPECT().GetChannel("user1").Return(nil),
		s.st.EXPECT().GetNick("user1").Return(nick1),
		// Send a random mode for an unknown channel
		s.st.EXPECT().GetChannel("#test2").Return(nil),
		s.st.EXPECT().GetNick("#test2").Return(nil),
	)
	c.h_MODE(parseLine(":user1!ident1@host1.com MODE user1 +w"))
	c.h_MODE(parseLine(":user1!ident1@host1.com MODE #test2 +is"))
}

// Test the handler for TOPIC messages
func TestTOPIC(t *testing.T) {
	c, s := setUp(t)
	defer s.tearDown()

	chan1 := state.NewChannel("#test1")

	// Assert that it has no topic originally
	if chan1.Topic != "" {
		t.Errorf("Test channel already has a topic.")
	}

	// Send a TOPIC line
	s.st.EXPECT().GetChannel("#test1").Return(chan1)
	c.h_TOPIC(parseLine(":user1!ident1@host1.com TOPIC #test1 :something something"))

	// Make sure the channel's topic has been changed
	if chan1.Topic != "something something" {
		t.Errorf("Topic of test channel not set correctly.")
	}

	// Check error paths -- send a topic for an unknown channel
	s.st.EXPECT().GetChannel("#test2").Return(nil)
	c.h_TOPIC(parseLine(":user1!ident1@host1.com TOPIC #test2 :dark side"))
}

// Test the handler for 311 / RPL_WHOISUSER
func Test311(t *testing.T) {
	c, s := setUp(t)
	defer s.tearDown()

	// Create user1, who we know little about
	nick1 := state.NewNick("user1")

	// Send a 311 reply
	s.st.EXPECT().GetNick("user1").Return(nick1)
	c.h_311(parseLine(":irc.server.org 311 test user1 ident1 host1.com * :name"))

	// Verify we now know more about user1
	if nick1.Ident != "ident1" ||
		nick1.Host != "host1.com" ||
		nick1.Name != "name" {
		t.Errorf("WHOIS info of user1 not set correctly.")
	}

	// Check error paths -- send a 311 for an unknown nick
	s.st.EXPECT().GetNick("user2").Return(nil)
	c.h_311(parseLine(":irc.server.org 311 test user2 ident2 host2.com * :dongs"))
}

// Test the handler for 324 / RPL_CHANNELMODEIS
func Test324(t *testing.T) {
	c, s := setUp(t)
	defer s.tearDown()

	// Create #test1, whose modes we don't know
	chan1 := state.NewChannel("#test1")

	// Send a 324 reply
	s.st.EXPECT().GetChannel("#test1").Return(chan1)
	c.h_324(parseLine(":irc.server.org 324 test #test1 +sk somekey"))
	if !chan1.Modes.Secret || chan1.Modes.Key != "somekey" {
		t.Errorf("Channel.ParseModes() not called correctly.")
	}

	// Check error paths -- send 324 for an unknown channel
	s.st.EXPECT().GetChannel("#test2").Return(nil)
	c.h_324(parseLine(":irc.server.org 324 test #test2 +pmt"))
}

// Test the handler for 332 / RPL_TOPIC
func Test332(t *testing.T) {
	c, s := setUp(t)
	defer s.tearDown()

	// Create #test1, whose topic we don't know
	chan1 := state.NewChannel("#test1")

	// Assert that it has no topic originally
	if chan1.Topic != "" {
		t.Errorf("Test channel already has a topic.")
	}

	// Send a 332 reply
	s.st.EXPECT().GetChannel("#test1").Return(chan1)
	c.h_332(parseLine(":irc.server.org 332 test #test1 :something something"))

	// Make sure the channel's topic has been changed
	if chan1.Topic != "something something" {
		t.Errorf("Topic of test channel not set correctly.")
	}

	// Check error paths -- send 332 for an unknown channel
	s.st.EXPECT().GetChannel("#test2").Return(nil)
	c.h_332(parseLine(":irc.server.org 332 test #test2 :dark side"))
}

// Test the handler for 352 / RPL_WHOREPLY
func Test352(t *testing.T) {
	c, s := setUp(t)
	defer s.tearDown()

	// Create user1, who we know little about
	nick1 := state.NewNick("user1")

	// Send a 352 reply
	s.st.EXPECT().GetNick("user1").Return(nick1)
	c.h_352(parseLine(":irc.server.org 352 test #test1 ident1 host1.com irc.server.org user1 G :0 name"))

	// Verify we now know more about user1
	if nick1.Ident != "ident1" ||
		nick1.Host != "host1.com" ||
		nick1.Name != "name" ||
		nick1.Modes.Invisible ||
		nick1.Modes.Oper {
		t.Errorf("WHO info of user1 not set correctly.")
	}

	// Check that modes are set correctly from WHOREPLY
	s.st.EXPECT().GetNick("user1").Return(nick1)
	c.h_352(parseLine(":irc.server.org 352 test #test1 ident1 host1.com irc.server.org user1 H* :0 name"))

	if !nick1.Modes.Invisible || !nick1.Modes.Oper {
		t.Errorf("WHO modes of user1 not set correctly.")
	}

	// Check error paths -- send a 352 for an unknown nick
	s.st.EXPECT().GetNick("user2").Return(nil)
	c.h_352(parseLine(":irc.server.org 352 test #test2 ident2 host2.com irc.server.org user2 G :0 fooo"))
}

// Test the handler for 353 / RPL_NAMREPLY
func Test353(t *testing.T) {
	c, s := setUp(t)
	defer s.tearDown()

	// Create #test1, whose user list we're mostly unfamiliar with
	chan1 := state.NewChannel("#test1")

	// Create maps for testing -- this is what the mock ST calls will return
	nicks := make(map[string]*state.Nick)
	privs := make(map[string]*state.ChanPrivs)

	nicks["test"] = c.Me
	privs["test"] = new(state.ChanPrivs)

	for _, n := range []string{"user1", "user2", "voice", "halfop",
		"op", "admin", "owner"} {
		nicks[n] = state.NewNick(n)
		privs[n] = new(state.ChanPrivs)
	}

	// 353 handler is called twice, so GetChannel will be called twice
	s.st.EXPECT().GetChannel("#test1").Return(chan1).Times(2)
	gomock.InOrder(
		// "test" is Me, i am known, and already on the channel
		s.st.EXPECT().GetNick("test").Return(c.Me),
		s.st.EXPECT().IsOn("#test1", "test").Return(privs["test"], true),
		// user1 is known, but not on the channel, so should be associated
		s.st.EXPECT().GetNick("user1").Return(nicks["user1"]),
		s.st.EXPECT().IsOn("#test1", "user1").Return(nil, false),
		s.st.EXPECT().Associate(chan1, nicks["user1"]).Return(privs["user1"]),
	)
	for _, n := range []string{"user2", "voice", "halfop", "op", "admin", "owner"} {
		gomock.InOrder(
			s.st.EXPECT().GetNick(n).Return(nil),
			s.st.EXPECT().NewNick(n).Return(nicks[n]),
			s.st.EXPECT().IsOn("#test1", n).Return(nil, false),
			s.st.EXPECT().Associate(chan1, nicks[n]).Return(privs[n]),
		)
	}

	// Send a couple of names replies (complete with trailing space)
	c.h_353(parseLine(":irc.server.org 353 test = #test1 :test @user1 user2 +voice "))
	c.h_353(parseLine(":irc.server.org 353 test = #test1 :%halfop @op &admin ~owner "))

	if p := privs["user2"]; p.Voice || p.HalfOp || p.Op || p.Admin || p.Owner {
		t.Errorf("353 handler incorrectly set modes on nick.")
	}

	if !privs["user1"].Op || !privs["voice"].Voice || !privs["halfop"].HalfOp ||
		!privs["op"].Op || !privs["admin"].Admin || !privs["owner"].Owner {
		t.Errorf("353 handler failed to set correct modes for nicks.")
	}

	// Check error paths -- send 353 for an unknown channel
	s.st.EXPECT().GetChannel("#test2").Return(nil)
	c.h_353(parseLine(":irc.server.org 353 test = #test2 :test ~user3"))
}

// Test the handler for 671 (unreal specific)
func Test671(t *testing.T) {
	c, s := setUp(t)
	defer s.tearDown()

	// Create user1, who should not be secure
	nick1 := state.NewNick("user1")
	if nick1.Modes.SSL {
		t.Errorf("Test nick user1 is already using SSL?")
	}

	// Send a 671 reply
	s.st.EXPECT().GetNick("user1").Return(nick1)
	c.h_671(parseLine(":irc.server.org 671 test user1 :some ignored text"))

	// Ensure user1 is now known to be on an SSL connection
	if !nick1.Modes.SSL {
		t.Errorf("Test nick user1 not using SSL?")
	}

	// Check error paths -- send a 671 for an unknown nick
	s.st.EXPECT().GetNick("user2").Return(nil)
	c.h_671(parseLine(":irc.server.org 671 test user2 :some ignored text"))
}
