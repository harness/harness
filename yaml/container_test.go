package yaml

import (
	"testing"

	"github.com/franela/goblin"
	"gopkg.in/yaml.v2"
)

func TestContainerNode(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("Containers", func() {
		g.Describe("given a yaml file", func() {

			g.It("should unmarshal", func() {
				in := []byte(sampleContainer)
				out := containerList{}
				err := yaml.Unmarshal(in, &out)
				if err != nil {
					g.Fail(err)
				}
				g.Assert(len(out.containers)).Equal(1)

				c := out.containers[0]
				g.Assert(c.Name).Equal("foo")
				g.Assert(c.Image).Equal("golang")
				g.Assert(c.Build).Equal(".")
				g.Assert(c.Pull).Equal(true)
				g.Assert(c.Detached).Equal(true)
				g.Assert(c.Privileged).Equal(true)
				g.Assert(c.Entrypoint).Equal([]string{"/bin/sh"})
				g.Assert(c.Command).Equal([]string{"yes"})
				g.Assert(c.Commands).Equal([]string{"whoami"})
				g.Assert(c.ExtraHosts).Equal([]string{"foo.com"})
				g.Assert(c.Volumes).Equal([]string{"/foo:/bar"})
				g.Assert(c.VolumesFrom).Equal([]string{"foo"})
				g.Assert(c.Devices).Equal([]string{"/dev/tty0"})
				g.Assert(c.Network).Equal("bridge")
				g.Assert(c.DNS).Equal([]string{"8.8.8.8"})
				g.Assert(c.MemSwapLimit).Equal(int64(1))
				g.Assert(c.MemLimit).Equal(int64(2))
				g.Assert(c.CPUQuota).Equal(int64(3))
				g.Assert(c.CPUSet).Equal("1,2")
				g.Assert(c.OomKillDisable).Equal(true)
				g.Assert(c.AuthConfig.Username).Equal("octocat")
				g.Assert(c.AuthConfig.Password).Equal("password")
				g.Assert(c.AuthConfig.Email).Equal("octocat@github.com")
				g.Assert(c.Vargs["access_key"]).Equal("970d28f4dd477bc184fbd10b376de753")
				g.Assert(c.Vargs["secret_key"]).Equal("9c5785d3ece6a9cdefa42eb99b58986f9095ff1c")
			})

			g.It("should unmarshal named", func() {
				in := []byte("foo: { name: bar }")
				out := containerList{}
				err := yaml.Unmarshal(in, &out)
				if err != nil {
					g.Fail(err)
				}
				g.Assert(len(out.containers)).Equal(1)
				g.Assert(out.containers[0].Name).Equal("bar")
			})

		})
	})
}

var sampleContainer = `
foo:
  image: golang
  build: .
  pull: true
  detach: true
  privileged: true
  environment:
    FOO: BAR
  entrypoint: /bin/sh
  command: "yes"
  commands: whoami
  extra_hosts: foo.com
  volumes: /foo:/bar
  volumes_from: foo
  devices: /dev/tty0
  network_mode: bridge
  dns: 8.8.8.8
  memswap_limit: 1
  mem_limit: 2
  cpu_quota: 3
  cpuset: 1,2
  oom_kill_disable: true

  auth_config:
    username: octocat
    password: password
    email: octocat@github.com

  access_key: 970d28f4dd477bc184fbd10b376de753
  secret_key: 9c5785d3ece6a9cdefa42eb99b58986f9095ff1c
`
