package state

import (
	"github.com/fluffle/golog/logging"
)

// The state manager interface
type StateTracker interface {
	// Nick methods
	NewNick(nick string) *Nick
	GetNick(nick string) *Nick
	ReNick(old, neu string)
	DelNick(nick string)
	// Channel methods
	NewChannel(channel string) *Channel
	GetChannel(channel string) *Channel
	DelChannel(channel string)
	// Information about ME!
	Me() *Nick
	// And the tracking operations
	IsOn(channel, nick string) (*ChanPrivs, bool)
	Associate(channel *Channel, nick *Nick) *ChanPrivs
	Dissociate(channel *Channel, nick *Nick)
	Wipe()
	// The state tracker can output a debugging string
	String() string
}

// ... and a struct to implement it ...
type stateTracker struct {
	// Map of channels we're on
	chans map[string]*Channel
	// Map of nicks we know about
	nicks map[string]*Nick

	// We need to keep state on who we are :-)
	me *Nick
}

// ... and a constructor to make it ...
func NewTracker(mynick string) *stateTracker {
	st := &stateTracker{
		chans: make(map[string]*Channel),
		nicks: make(map[string]*Nick),
	}
	st.me = st.NewNick(mynick)
	return st
}

// ... and a method to wipe the state clean.
func (st *stateTracker) Wipe() {
	// Deleting all the channels implicitly deletes every nick but me.
	for _, ch := range st.chans {
		st.delChannel(ch)
	}
}

/******************************************************************************\
 * tracker methods to create/look up nicks/channels
\******************************************************************************/

// Creates a new Nick, initialises it, and stores it so it
// can be properly tracked for state management purposes.
func (st *stateTracker) NewNick(n string) *Nick {
	if _, ok := st.nicks[n]; ok {
		logging.Warn("StateTracker.NewNick(): %s already tracked.", n)
		return nil
	}
	st.nicks[n] = NewNick(n)
	return st.nicks[n]
}

// Returns a Nick for the nick n, if we're tracking it.
func (st *stateTracker) GetNick(n string) *Nick {
	if nk, ok := st.nicks[n]; ok {
		return nk
	}
	return nil
}

// Signals to the tracker that a Nick should be tracked
// under a "neu" nick rather than the old one.
func (st *stateTracker) ReNick(old, neu string) {
	if nk, ok := st.nicks[old]; ok {
		if _, ok := st.nicks[neu]; !ok {
			nk.Nick = neu
			delete(st.nicks, old)
			st.nicks[neu] = nk
			for ch, _ := range nk.chans {
				// We also need to update the lookup maps of all the channels
				// the nick is on, to keep things in sync.
				delete(ch.lookup, old)
				ch.lookup[neu] = nk
			}
		} else {
			logging.Warn("StateTracker.ReNick(): %s already exists.", neu)
		}
	} else {
		logging.Warn("StateTracker.ReNick(): %s not tracked.", old)
	}
}

// Removes a Nick from being tracked.
func (st *stateTracker) DelNick(n string) {
	if nk, ok := st.nicks[n]; ok {
		if nk != st.me {
			st.delNick(nk)
		} else {
			logging.Warn("StateTracker.DelNick(): won't delete myself.")
		}
	} else {
		logging.Warn("StateTracker.DelNick(): %s not tracked.", n)
	}
}

func (st *stateTracker) delNick(nk *Nick) {
	if nk == st.me {
		// Shouldn't get here => internal state tracking code is fubar.
		logging.Error("StateTracker.DelNick(): TRYING TO DELETE ME :-(")
		return
	}
	delete(st.nicks, nk.Nick)
	for ch, _ := range nk.chans {
		nk.delChannel(ch)
		ch.delNick(nk)
		if len(ch.nicks) == 0 {
			// Deleting a nick from tracking shouldn't empty any channels as
			// *we* should be on the channel with them to be tracking them.
			logging.Error("StateTracker.delNick(): deleting nick %s emptied "+
				"channel %s, this shouldn't happen!", nk.Nick, ch.Name)
		}
	}
}

// Creates a new Channel, initialises it, and stores it so it
// can be properly tracked for state management purposes.
func (st *stateTracker) NewChannel(c string) *Channel {
	if _, ok := st.chans[c]; ok {
		logging.Warn("StateTracker.NewChannel(): %s already tracked.", c)
		return nil
	}
	st.chans[c] = NewChannel(c)
	return st.chans[c]
}

// Returns a Channel for the channel c, if we're tracking it.
func (st *stateTracker) GetChannel(c string) *Channel {
	if ch, ok := st.chans[c]; ok {
		return ch
	}
	return nil
}

// Removes a Channel from being tracked.
func (st *stateTracker) DelChannel(c string) {
	if ch, ok := st.chans[c]; ok {
		st.delChannel(ch)
	} else {
		logging.Warn("StateTracker.DelChannel(): %s not tracked.", c)
	}
}

func (st *stateTracker) delChannel(ch *Channel) {
	delete(st.chans, ch.Name)
	for nk, _ := range ch.nicks {
		ch.delNick(nk)
		nk.delChannel(ch)
		if len(nk.chans) == 0 && nk != st.me {
			// We're no longer in any channels with this nick.
			st.delNick(nk)
		}
	}
}

// Returns the Nick the state tracker thinks is Me.
func (st *stateTracker) Me() *Nick {
	return st.me
}

// Returns true if both the channel c and the nick n are tracked
// and the nick is associated with the channel.
func (st *stateTracker) IsOn(c, n string) (*ChanPrivs, bool) {
	nk := st.GetNick(n)
	ch := st.GetChannel(c)
	if nk != nil && ch != nil {
		return nk.IsOn(ch)
	}
	return nil, false
}

// Associates an already known nick with an already known channel.
func (st *stateTracker) Associate(ch *Channel, nk *Nick) *ChanPrivs {
	if ch == nil || nk == nil {
		logging.Error("StateTracker.Associate(): passed nil values :-(")
		return nil
	} else if _ch, ok := st.chans[ch.Name]; !ok || ch != _ch {
		// As we can implicitly delete both nicks and channels from being
		// tracked by dissociating one from the other, we should verify that
		// we're not being passed an old Nick or Channel.
		logging.Error("StateTracker.Associate(): channel %s not found in "+
			"(or differs from) internal state.", ch.Name)
		return nil
	} else if _nk, ok := st.nicks[nk.Nick]; !ok || nk != _nk {
		logging.Error("StateTracker.Associate(): nick %s not found in "+
			"(or differs from) internal state.", nk.Nick)
		return nil
	} else if _, ok := nk.IsOn(ch); ok {
		logging.Warn("StateTracker.Associate(): %s already on %s.",
			nk.Nick, ch.Name)
		return nil
	}
	cp := new(ChanPrivs)
	ch.addNick(nk, cp)
	nk.addChannel(ch, cp)
	return cp
}

// Dissociates an already known nick from an already known channel.
// Does some tidying up to stop tracking nicks we're no longer on
// any common channels with, and channels we're no longer on.
func (st *stateTracker) Dissociate(ch *Channel, nk *Nick) {
	if ch == nil || nk == nil {
		logging.Error("StateTracker.Dissociate(): passed nil values :-(")
	} else if _ch, ok := st.chans[ch.Name]; !ok || ch != _ch {
		// As we can implicitly delete both nicks and channels from being
		// tracked by dissociating one from the other, we should verify that
		// we're not being passed an old Nick or Channel.
		logging.Error("StateTracker.Dissociate(): channel %s not found in "+
			"(or differs from) internal state.", ch.Name)
	} else if _nk, ok := st.nicks[nk.Nick]; !ok || nk != _nk {
		logging.Error("StateTracker.Dissociate(): nick %s not found in "+
			"(or differs from) internal state.", nk.Nick)
	} else if _, ok := nk.IsOn(ch); !ok {
		logging.Warn("StateTracker.Dissociate(): %s not on %s.",
			nk.Nick, ch.Name)
	} else if nk == st.me {
		// I'm leaving the channel for some reason, so it won't be tracked.
		st.delChannel(ch)
	} else {
		// Remove the nick from the channel and the channel from the nick.
		ch.delNick(nk)
		nk.delChannel(ch)
		if len(nk.chans) == 0 {
			// We're no longer in any channels with this nick.
			st.delNick(nk)
		}
	}
}

func (st *stateTracker) String() string {
	str := "GoIRC Channels\n"
	str += "--------------\n\n"
	for _, ch := range st.chans {
		str += ch.String() + "\n"
	}
	str += "GoIRC NickNames\n"
	str += "---------------\n\n"
	for _, n := range st.nicks {
		if n != st.me {
			str += n.String() + "\n"
		}
	}
	return str
}
