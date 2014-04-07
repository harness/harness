package client

import "testing"

func TestClientCommands(t *testing.T) {
	c, s := setUp(t)
	defer s.tearDown()

	c.Pass("password")
	s.nc.Expect("PASS password")

	c.Nick("test")
	s.nc.Expect("NICK test")

	c.User("test", "Testing IRC")
	s.nc.Expect("USER test 12 * :Testing IRC")

	c.Raw("JUST a raw :line")
	s.nc.Expect("JUST a raw :line")

	c.Join("#foo")
	s.nc.Expect("JOIN #foo")

	c.Part("#foo")
	s.nc.Expect("PART #foo")
	c.Part("#foo", "Screw you guys...")
	s.nc.Expect("PART #foo :Screw you guys...")

	c.Quit()
	s.nc.Expect("QUIT :GoBye!")
	c.Quit("I'm going home.")
	s.nc.Expect("QUIT :I'm going home.")

	c.Whois("somebody")
	s.nc.Expect("WHOIS somebody")

	c.Who("*@some.host.com")
	s.nc.Expect("WHO *@some.host.com")

	c.Privmsg("#foo", "bar")
	s.nc.Expect("PRIVMSG #foo :bar")

	c.Notice("somebody", "something")
	s.nc.Expect("NOTICE somebody :something")

	c.Ctcp("somebody", "ping", "123456789")
	s.nc.Expect("PRIVMSG somebody :\001PING 123456789\001")

	c.CtcpReply("somebody", "pong", "123456789")
	s.nc.Expect("NOTICE somebody :\001PONG 123456789\001")

	c.Version("somebody")
	s.nc.Expect("PRIVMSG somebody :\001VERSION\001")

	c.Action("#foo", "pokes somebody")
	s.nc.Expect("PRIVMSG #foo :\001ACTION pokes somebody\001")

	c.Topic("#foo")
	s.nc.Expect("TOPIC #foo")
	c.Topic("#foo", "la la la")
	s.nc.Expect("TOPIC #foo :la la la")

	c.Mode("#foo")
	s.nc.Expect("MODE #foo")
	c.Mode("#foo", "+o somebody")
	s.nc.Expect("MODE #foo +o somebody")

	c.Away()
	s.nc.Expect("AWAY")
	c.Away("Dave's not here, man.")
	s.nc.Expect("AWAY :Dave's not here, man.")

	c.Invite("somebody", "#foo")
	s.nc.Expect("INVITE somebody #foo")

	c.Oper("user", "pass")
	s.nc.Expect("OPER user pass")
}
