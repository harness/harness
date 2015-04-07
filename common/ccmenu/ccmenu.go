package ccmenu

import (
	"encoding/xml"
	"strconv"
	"time"

	"github.com/drone/drone/common"
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

func NewCC(r *common.Repo, b *common.Build, url string) *CCProjects {
	proj := &CCProject{
		Name:            r.Owner + "/" + r.Name,
		WebURL:          url,
		Activity:        "Building",
		LastBuildStatus: "Unknown",
		LastBuildLabel:  "Unknown",
	}

	// if the build is not currently running then
	// we can return the latest build status.
	if b.State != common.StatePending &&
		b.State != common.StateRunning {
		proj.Activity = "Sleeping"
		proj.LastBuildTime = time.Unix(b.Started, 0).Format(time.RFC3339)
		proj.LastBuildLabel = strconv.Itoa(b.Number)
	}

	// ensure the last build state accepts a valid
	// ccmenu enumeration
	switch b.State {
	case common.StateError, common.StateKilled:
		proj.LastBuildStatus = "Exception"
	case common.StateSuccess:
		proj.LastBuildStatus = "Success"
	case common.StateFailure:
		proj.LastBuildStatus = "Failure"
	default:
		proj.LastBuildStatus = "Unknown"
	}

	return &CCProjects{Project: proj}
}
