package state

import (
	"fmt"
	"github.com/fluffle/golog/logging"
	"reflect"
	"strconv"
)

// A struct representing an IRC channel
type Channel struct {
	Name, Topic string
	Modes       *ChanMode
	lookup      map[string]*Nick
	nicks       map[*Nick]*ChanPrivs
}

// A struct representing the modes of an IRC Channel
// (the ones we care about, at least).
// http://www.unrealircd.com/files/docs/unreal32docs.html#userchannelmodes
type ChanMode struct {
	// MODE +p, +s, +t, +n, +m
	Private, Secret, ProtectedTopic, NoExternalMsg, Moderated bool

	// MODE +i, +O, +z
	InviteOnly, OperOnly, SSLOnly bool

	// MODE +r, +Z
	Registered, AllSSL bool

	// MODE +k
	Key string

	// MODE +l
	Limit int
}

// A struct representing the modes a Nick can have on a Channel
type ChanPrivs struct {
	// MODE +q, +a, +o, +h, +v
	Owner, Admin, Op, HalfOp, Voice bool
}

// Map ChanMode fields to IRC mode characters
var StringToChanMode = map[string]string{}
var ChanModeToString = map[string]string{
	"Private":        "p",
	"Secret":         "s",
	"ProtectedTopic": "t",
	"NoExternalMsg":  "n",
	"Moderated":      "m",
	"InviteOnly":     "i",
	"OperOnly":       "O",
	"SSLOnly":        "z",
	"Registered":     "r",
	"AllSSL":         "Z",
	"Key":            "k",
	"Limit":          "l",
}

// Map *irc.ChanPrivs fields to IRC mode characters
var StringToChanPriv = map[string]string{}
var ChanPrivToString = map[string]string{
	"Owner":  "q",
	"Admin":  "a",
	"Op":     "o",
	"HalfOp": "h",
	"Voice":  "v",
}

// Map *irc.ChanPrivs fields to the symbols used to represent these modes
// in NAMES and WHOIS responses
var ModeCharToChanPriv = map[byte]string{}
var ChanPrivToModeChar = map[string]byte{
	"Owner":  '~',
	"Admin":  '&',
	"Op":     '@',
	"HalfOp": '%',
	"Voice":  '+',
}

// Init function to fill in reverse mappings for *toString constants.
func init() {
	for k, v := range ChanModeToString {
		StringToChanMode[v] = k
	}
	for k, v := range ChanPrivToString {
		StringToChanPriv[v] = k
	}
	for k, v := range ChanPrivToModeChar {
		ModeCharToChanPriv[v] = k
	}
}

/******************************************************************************\
 * Channel methods for state management
\******************************************************************************/

func NewChannel(name string) *Channel {
	return &Channel{
		Name:   name,
		Modes:  new(ChanMode),
		nicks:  make(map[*Nick]*ChanPrivs),
		lookup: make(map[string]*Nick),
	}
}

// Returns true if the Nick is associated with the Channel
func (ch *Channel) IsOn(nk *Nick) (*ChanPrivs, bool) {
	cp, ok := ch.nicks[nk]
	return cp, ok
}

func (ch *Channel) IsOnStr(n string) (*Nick, bool) {
	nk, ok := ch.lookup[n]
	return nk, ok
}

// Associates a Nick with a Channel
func (ch *Channel) addNick(nk *Nick, cp *ChanPrivs) {
	if _, ok := ch.nicks[nk]; !ok {
		ch.nicks[nk] = cp
		ch.lookup[nk.Nick] = nk
	} else {
		logging.Warn("Channel.addNick(): %s already on %s.", nk.Nick, ch.Name)
	}
}

// Disassociates a Nick from a Channel.
func (ch *Channel) delNick(nk *Nick) {
	if _, ok := ch.nicks[nk]; ok {
		delete(ch.nicks, nk)
		delete(ch.lookup, nk.Nick)
	} else {
		logging.Warn("Channel.delNick(): %s not on %s.", nk.Nick, ch.Name)
	}
}

// Parses mode strings for a channel.
func (ch *Channel) ParseModes(modes string, modeargs ...string) {
	var modeop bool // true => add mode, false => remove mode
	var modestr string
	for i := 0; i < len(modes); i++ {
		switch m := modes[i]; m {
		case '+':
			modeop = true
			modestr = string(m)
		case '-':
			modeop = false
			modestr = string(m)
		case 'i':
			ch.Modes.InviteOnly = modeop
		case 'm':
			ch.Modes.Moderated = modeop
		case 'n':
			ch.Modes.NoExternalMsg = modeop
		case 'p':
			ch.Modes.Private = modeop
		case 'r':
			ch.Modes.Registered = modeop
		case 's':
			ch.Modes.Secret = modeop
		case 't':
			ch.Modes.ProtectedTopic = modeop
		case 'z':
			ch.Modes.SSLOnly = modeop
		case 'Z':
			ch.Modes.AllSSL = modeop
		case 'O':
			ch.Modes.OperOnly = modeop
		case 'k':
			if modeop && len(modeargs) != 0 {
				ch.Modes.Key, modeargs = modeargs[0], modeargs[1:]
			} else if !modeop {
				ch.Modes.Key = ""
			} else {
				logging.Warn("Channel.ParseModes(): not enough arguments to "+
					"process MODE %s %s%c", ch.Name, modestr, m)
			}
		case 'l':
			if modeop && len(modeargs) != 0 {
				ch.Modes.Limit, _ = strconv.Atoi(modeargs[0])
				modeargs = modeargs[1:]
			} else if !modeop {
				ch.Modes.Limit = 0
			} else {
				logging.Warn("Channel.ParseModes(): not enough arguments to "+
					"process MODE %s %s%c", ch.Name, modestr, m)
			}
		case 'q', 'a', 'o', 'h', 'v':
			if len(modeargs) != 0 {
				if nk, ok := ch.lookup[modeargs[0]]; ok {
					cp := ch.nicks[nk]
					switch m {
					case 'q':
						cp.Owner = modeop
					case 'a':
						cp.Admin = modeop
					case 'o':
						cp.Op = modeop
					case 'h':
						cp.HalfOp = modeop
					case 'v':
						cp.Voice = modeop
					}
					modeargs = modeargs[1:]
				} else {
					logging.Warn("Channel.ParseModes(): untracked nick %s "+
						"received MODE on channel %s", modeargs[0], ch.Name)
				}
			} else {
				logging.Warn("Channel.ParseModes(): not enough arguments to "+
					"process MODE %s %s%c", ch.Name, modestr, m)
			}
		default:
			logging.Info("Channel.ParseModes(): unknown mode char %c", m)
		}
	}
}

// Nicks returns a list of *Nick that are on the channel.
func (ch *Channel) Nicks() []*Nick {
	nicks := make([]*Nick, 0, len(ch.lookup))
	for _, nick := range ch.lookup {
		nicks = append(nicks, nick)
	}
	return nicks
}

// NicksStr returns a list of nick strings that are on the channel.
func (ch *Channel) NicksStr() []string {
	nicks := make([]string, 0, len(ch.lookup))
	for _, nick := range ch.lookup {
		nicks = append(nicks, nick.Nick)
	}
	return nicks
}

// Returns a string representing the channel. Looks like:
//	Channel: <channel name> e.g. #moo
//	Topic: <channel topic> e.g. Discussing the merits of cows!
//	Mode: <channel modes> e.g. +nsti
//	Nicks:
//		<nick>: <privs> e.g. CowMaster: +o
//		...
func (ch *Channel) String() string {
	str := "Channel: " + ch.Name + "\n\t"
	str += "Topic: " + ch.Topic + "\n\t"
	str += "Modes: " + ch.Modes.String() + "\n\t"
	str += "Nicks: \n"
	for nk, cp := range ch.nicks {
		str += "\t\t" + nk.Nick + ": " + cp.String() + "\n"
	}
	return str
}

// Returns a string representing the channel modes. Looks like:
//	+npk key
func (cm *ChanMode) String() string {
	str := "+"
	a := make([]string, 0)
	v := reflect.Indirect(reflect.ValueOf(cm))
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		switch f := v.Field(i); f.Kind() {
		case reflect.Bool:
			if f.Bool() {
				str += ChanModeToString[t.Field(i).Name]
			}
		case reflect.String:
			if f.String() != "" {
				str += ChanModeToString[t.Field(i).Name]
				a = append(a, f.String())
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if f.Int() != 0 {
				str += ChanModeToString[t.Field(i).Name]
				a = append(a, fmt.Sprintf("%d", f.Int()))
			}
		}
	}
	for _, s := range a {
		if s != "" {
			str += " " + s
		}
	}
	if str == "+" {
		str = "No modes set"
	}
	return str
}

// Returns a string representing the channel privileges. Looks like:
//	+o
func (cp *ChanPrivs) String() string {
	str := "+"
	v := reflect.Indirect(reflect.ValueOf(cp))
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		switch f := v.Field(i); f.Kind() {
		// only bools here at the mo too!
		case reflect.Bool:
			if f.Bool() {
				str += ChanPrivToString[t.Field(i).Name]
			}
		}
	}
	if str == "+" {
		str = "No modes set"
	}
	return str
}
