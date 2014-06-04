package model

import (
	"fmt"
	"time"
)

const (
	StatusNone    = "None"
	StatusEnqueue = "Pending"
	StatusStarted = "Started"
	StatusSuccess = "Success"
	StatusFailure = "Failure"
	StatusError   = "Error"
)

type Build struct {
	ID          int64     `meddler:"id,pk"            json:"id"`
	CommitID    int64     `meddler:"commit_id"        json:"-"`
	Slug        string    `meddler:"slug"             json:"slug"`
	Status      string    `meddler:"status"           json:"status"`
	Started     time.Time `meddler:"started,utctime"  json:"started"`
	Finished    time.Time `meddler:"finished,utctime" json:"finished"`
	Duration    int64     `meddler:"duration"         json:"duration"`
	Created     time.Time `meddler:"created,utctime"  json:"created"`
	Updated     time.Time `meddler:"updated,utctime"  json:"updated"`
	Stdout      string    `meddler:"stdout"           json:"-"`
	BuildScript string    `meddler:"buildscript"      json:"-"`
}

// HumanDuration returns a human-readable approximation of a duration
// (eg. "About a minute", "4 hours ago", etc.)
func (b *Build) HumanDuration() string {
	d := time.Duration(b.Duration)
	if seconds := int(d.Seconds()); seconds < 1 {
		return "Less than a second"
	} else if seconds < 60 {
		return fmt.Sprintf("%d seconds", seconds)
	} else if minutes := int(d.Minutes()); minutes == 1 {
		return "About a minute"
	} else if minutes < 60 {
		return fmt.Sprintf("%d minutes", minutes)
	} else if hours := int(d.Hours()); hours == 1 {
		return "About an hour"
	} else if hours < 48 {
		return fmt.Sprintf("%d hours", hours)
	} else if hours < 24*7*2 {
		return fmt.Sprintf("%d days", hours/24)
	} else if hours < 24*30*3 {
		return fmt.Sprintf("%d weeks", hours/24/7)
	} else if hours < 24*365*2 {
		return fmt.Sprintf("%d months", hours/24/30)
	}
	return fmt.Sprintf("%f years", d.Hours()/24/365)
}

// Returns the Started Date as an ISO8601
// formatted string.
func (b *Build) StartedString() string {
	return b.Started.Format("2006-01-02T15:04:05Z")
}

// Returns the Started Date as an ISO8601
// formatted string.
func (b *Build) FinishedString() string {
	return b.Finished.Format("2006-01-02T15:04:05Z")
}

// Returns true if the Build statis is Started
// or Pending, indicating it is currently running.
func (b *Build) IsRunning() bool {
	return (b.Status == StatusStarted || b.Status == StatusEnqueue)
}
