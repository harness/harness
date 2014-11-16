package deploy

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/drone/drone/plugin/condition"
	"github.com/drone/drone/shared/build/buildfile"
)

// SSH struct holds configuration data for deployment
// via ssh, deployment done by doing rsync on the file(s) listed
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
	// to the target host and always relative to the project's root directory,
	// like so:
	//   artifacts: ./
	// or
	//   artifacts: build/
	//
	// To list multiple artifacts, please use multiple-line string
	// directive instead
	// eg.
	//    artifacts: |
	//      bin/
	//      include/
	//      lib/
	Artifacts string `yaml:"artifacts,omitempty"`

	// Cmd is the command executed at target host after the artifacts
	// is deployed.
	// To use multiple commands you can write it naturally with
	// multiple-line string directive
	// eg.
	//    cmd: |
	//      git clone github.com/myproject/myrepo
	//      cd myrepo
	//      bundle install --deployment
	//      bundle exec rake assets:precompile
	//      touch tmp/restart.txt
	Cmd string `yaml:"cmd,omitempty"`

	Condition *condition.Condition `yaml:"when,omitempty"`
}

// Write down the buildfile
func (s *SSH) Write(f *buildfile.Buildfile) {
	rsyncTemplate := "rsync -avze 'ssh -p %s' --files-from ${ARTIFACTS} ./ %s"
	cmdTemplate := "printf %q > /tmp/drone_deploy; sh /tmp/drone_deploy; rm -f /tmp/drone_deploy"

	host := strings.SplitN(s.Target, " ", 2)
	if len(host) == 1 {
		host = append(host, "22")
	}
	if _, err := strconv.Atoi(host[1]); err != nil {
		host[1] = "22"
	}

	// Back to the project's top directory, just in case
	f.WriteCmdSilent("cd ${DRONE_BUILD_DIR}")

	if len(s.Artifacts) > 0 {
		f.WriteEnv("ARTIFACTS", "$(mktemp)")
		f.WriteCmdSilent(fmt.Sprintf("printf %q > ${ARTIFACTS}", s.Artifacts))
		f.WriteCmd(fmt.Sprintf(rsyncTemplate, host[1], host[0]))
		f.WriteCmdSilent("rm -f ${ARTIFACTS}")
	}

	if len(s.Cmd) > 0 {
		sshCmd := "ssh -o StrictHostKeyChecking=no -p %s %s %q"
		hostnpath := strings.SplitN(host[0], ":", 2)
		if len(hostnpath) == 2 {
			// ensure script to run under target directory
			s.Cmd = fmt.Sprintf("cd %s\n%s", hostnpath[1], s.Cmd)
		}
		cmd := fmt.Sprintf(cmdTemplate, s.Cmd)
		f.WriteCmdSilent(fmt.Sprintf(sshCmd, host[1], hostnpath[0], cmd))
	}
}

func (s *SSH) GetCondition() *condition.Condition {
	return s.Condition
}
