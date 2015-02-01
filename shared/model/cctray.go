package model

import (
	"encoding/xml"
	"time"
)

type CCProjects struct {
	XMLName xml.Name   `xml:"Projects"`
	Project *CCProject `xml:"Project"`
}

type CCProject struct {
	XMLName         xml.Name `xml:"Project"`
	Name            string   `xml:"name,attr"`
	Activity        string   `xml:"activity,attr"`
	LastBuildStatus string   `xml:"lastBuildStatus,attr"`
	LastBuildLabel  string   `xml:"lastBuildLabel,attr"`
	LastBuildTime   string   `xml:"lastBuildTime,attr"`
	WebURL          string   `xml:"webUrl,attr"`
}

func NewCC(r *Repo, c *Commit, url string) *CCProjects {
	proj := &CCProject{
		Name:            r.Owner + "/" + r.Name,
		WebURL:          url,
		Activity:        "Building",
		LastBuildStatus: "Unknown",
		LastBuildLabel:  "Unknown",
	}

	// if the build is not currently running then
	// we can return the latest build status.
	if c.Status != StatusStarted &&
		c.Status != StatusEnqueue {
		proj.Activity = "Sleeping"
		proj.LastBuildStatus = c.Status
		proj.LastBuildTime = time.Unix(c.Started, 0).Format(time.RFC3339)
		proj.LastBuildLabel = c.ShaShort()
	}

	// If the build is not running, and not successful,
	// then set to Failure. Not sure CCTray will support
	// our custom failure types (ie Killed)
	if c.Status != StatusStarted &&
		c.Status != StatusEnqueue &&
		c.Status != StatusSuccess {
		proj.LastBuildStatus = StatusFailure
	}

	return &CCProjects{Project: proj}
}
