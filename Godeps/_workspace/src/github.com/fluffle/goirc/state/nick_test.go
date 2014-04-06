package state

import (
	"testing"
)

func TestNewNick(t *testing.T) {
	nk := NewNick("test1")

	if nk.Nick != "test1" {
		t.Errorf("Nick not created correctly by NewNick()")
	}
	if len(nk.chans) != 0 || len(nk.lookup) != 0 {
		t.Errorf("Nick maps contain data after NewNick()")
	}
}

func TestAddChannel(t *testing.T) {
	nk := NewNick("test1")
	ch := NewChannel("#test1")
	cp := new(ChanPrivs)

	nk.addChannel(ch, cp)

	if len(nk.chans) != 1 || len(nk.lookup) != 1 {
		t.Errorf("Channel lists not updated correctly for add.")
	}
	if c, ok := nk.chans[ch]; !ok || c != cp {
		t.Errorf("Channel #test1 not properly stored in chans map.")
	}
	if c, ok := nk.lookup["#test1"]; !ok || c != ch {
		t.Errorf("Channel #test1 not properly stored in lookup map.")
	}
}

func TestDelChannel(t *testing.T) {
	nk := NewNick("test1")
	ch := NewChannel("#test1")
	cp := new(ChanPrivs)

	nk.addChannel(ch, cp)
	nk.delChannel(ch)
	if len(nk.chans) != 0 || len(nk.lookup) != 0 {
		t.Errorf("Channel lists not updated correctly for del.")
	}
	if c, ok := nk.chans[ch]; ok || c != nil {
		t.Errorf("Channel #test1 not properly removed from chans map.")
	}
	if c, ok := nk.lookup["#test1"]; ok || c != nil {
		t.Errorf("Channel #test1 not properly removed from lookup map.")
	}
}

func TestNickParseModes(t *testing.T) {
	nk := NewNick("test1")
	md := nk.Modes

	// Modes should all be false for a new nick
	if md.Invisible || md.Oper || md.WallOps || md.HiddenHost || md.SSL {
		t.Errorf("Modes for new nick set to true.")
	}

	// Set a couple of modes, for testing.
	md.Invisible = true
	md.HiddenHost = true

	// Parse a mode line that flips one true to false and two false to true
	nk.ParseModes("+z-x+w")

	if !md.Invisible || md.Oper || !md.WallOps || md.HiddenHost || !md.SSL {
		t.Errorf("Modes not flipped correctly by ParseModes.")
	}
}
