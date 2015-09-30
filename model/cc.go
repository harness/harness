package model

import (
	"encoding/xml"
	"strconv"
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

func NewCC(r *Repo, b *Build, link string) *CCProjects {
	proj := &CCProject{
		Name:            r.FullName,
		WebURL:          link,
		Activity:        "Building",
		LastBuildStatus: "Unknown",
		LastBuildLabel:  "Unknown",
	}

	// if the build is not currently running then
	// we can return the latest build status.
	if b.Status != StatusPending &&
		b.Status != StatusRunning {
		proj.Activity = "Sleeping"
		proj.LastBuildTime = time.Unix(b.Started, 0).Format(time.RFC3339)
		proj.LastBuildLabel = strconv.Itoa(b.Number)
	}

	// ensure the last build Status accepts a valid
	// ccmenu enumeration
	switch b.Status {
	case StatusError, StatusKilled:
		proj.LastBuildStatus = "Exception"
	case StatusSuccess:
		proj.LastBuildStatus = "Success"
	case StatusFailure:
		proj.LastBuildStatus = "Failure"
	}

	return &CCProjects{Project: proj}
}
