package api

import (
	"bytes"
	"strconv"
	"time"
)

const (
	// GitTimeLayout is the (default) time layout used by git.
	GitTimeLayout = "Mon Jan _2 15:04:05 2006 -0700"
)

// Helper to get a signature from the commit line, which looks like these:
//
//	author Patrick Gundlach <gundlach@speedata.de> 1378823654 +0200
//	author Patrick Gundlach <gundlach@speedata.de> Thu, 07 Apr 2005 22:13:13 +0200
//
// but without the "author " at the beginning (this method should)
// be used for author and committer.
func NewSignatureFromCommitLine(line []byte) (sig *Signature, err error) {
	sig = new(Signature)
	emailStart := bytes.LastIndexByte(line, '<')
	emailEnd := bytes.LastIndexByte(line, '>')
	if emailStart == -1 || emailEnd == -1 || emailEnd < emailStart {
		return
	}

	sig.Identity.Name = string(line[:emailStart-1])
	sig.Identity.Email = string(line[emailStart+1 : emailEnd])

	hasTime := emailEnd+2 < len(line)
	if !hasTime {
		return
	}

	// Check date format.
	firstChar := line[emailEnd+2]
	if firstChar >= 48 && firstChar <= 57 {
		idx := bytes.IndexByte(line[emailEnd+2:], ' ')
		if idx < 0 {
			return
		}

		timestring := string(line[emailEnd+2 : emailEnd+2+idx])
		seconds, _ := strconv.ParseInt(timestring, 10, 64)
		sig.When = time.Unix(seconds, 0)

		idx += emailEnd + 3
		if idx >= len(line) || idx+5 > len(line) {
			return
		}

		timezone := string(line[idx : idx+5])
		tzhours, err1 := strconv.ParseInt(timezone[0:3], 10, 64)
		tzmins, err2 := strconv.ParseInt(timezone[3:], 10, 64)
		if err1 != nil || err2 != nil {
			return
		}
		if tzhours < 0 {
			tzmins *= -1
		}
		tz := time.FixedZone("", int(tzhours*60*60+tzmins*60))
		sig.When = sig.When.In(tz)
	} else {
		sig.When, err = time.Parse(GitTimeLayout, string(line[emailEnd+2:]))
		if err != nil {
			return
		}
	}
	return sig, err
}
