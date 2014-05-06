package deploy

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/drone/drone/pkg/build/buildfile"
)

// SSH struct holds configuration data for deployment
// via ssh, deployment done by scp-ing file(s) listed
// in artifacts to the target host, and then run cmd
// remotely.
// It is assumed that the target host already
// add this repo public key in the host's `authorized_hosts`
// file. And the private key is already copied to `.ssh/id_rsa`
// inside the build container. No further check will be done.
type SSH struct {

	// Target is the deployment host in this format
	//   user@hostname:/full/path <PORT>
	//
	// PORT may be omitted if its default to port 22.
	Target string `yaml:"target,omitempty"`

	// Artifacts is a list of files/dirs to be deployed
	// to the target host. If artifacts list more than one file
	// it will be compressed into a single tar.gz file.
	// if artifacts contain:
	//   - GITARCHIVE
	//
	// other file listed in artifacts will be ignored, instead, we will
	// create git archive from the current revision and deploy that file
	// alone.
	// If you need to deploy the git archive along with some other files,
	// please use build script to create the git archive, and then list
	// the archive name here with the other files.
	Artifacts []string `yaml:"artifacts,omitempty"`

	// Cmd is a single command executed at target host after the artifacts
	// is deployed.
	Cmd string `yaml:"cmd,omitempty"`
}

// Write down the buildfile
func (s *SSH) Write(f *buildfile.Buildfile) {
	host := strings.SplitN(s.Target, " ", 2)
	if len(host) == 1 {
		host = append(host, "22")
	}
	if _, err := strconv.Atoi(host[1]); err != nil {
		host[1] = "22"
	}

	// Is artifact created?
	artifact := false

	for _, a := range s.Artifacts {
		if a == "GITARCHIVE" {
			artifact = createGitArchive(f)
			break
		}
	}

	if !artifact {
		if len(s.Artifacts) > 1 {
			artifact = compress(f, s.Artifacts)
		} else if len(s.Artifacts) == 1 {
			f.WriteEnv("ARTIFACT", s.Artifacts[0])
			artifact = true
		}
	}

	if artifact {
		scpCmd := "scp -o StrictHostKeyChecking=no -P %s -r ${ARTIFACT} %s"
		f.WriteCmd(fmt.Sprintf(scpCmd, host[1], host[0]))
	}

	if len(s.Cmd) > 0 {
		sshCmd := "ssh -o StrictHostKeyChecking=no -p %s %s %s"
		f.WriteCmd(fmt.Sprintf(sshCmd, host[1], strings.SplitN(host[0], ":", 2)[0], s.Cmd))
	}
}

func createGitArchive(f *buildfile.Buildfile) bool {
	f.WriteEnv("COMMIT", "$(git rev-parse HEAD)")
	f.WriteEnv("ARTIFACT", "${PWD##*/}-${COMMIT}.tar.gz")
	f.WriteCmdSilent("git archive --format=tar.gz --prefix=${PWD##*/}/ ${COMMIT} > ${ARTIFACT}")
	return true
}

func compress(f *buildfile.Buildfile, files []string) bool {
	cmd := "tar -cf ${ARTIFACT} %s"
	f.WriteEnv("ARTIFACT", "${PWD##*/}.tar.gz")
	f.WriteCmdSilent(fmt.Sprintf(cmd, strings.Join(files, " ")))
	return true
}
