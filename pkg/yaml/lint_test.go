package parser

import (
	"testing"

	common "github.com/drone/drone/pkg/types"
	"github.com/franela/goblin"
)

func Test_Linter(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Linter", func() {

		g.It("Should fail when nil build", func() {
			c := &common.Config{}
			g.Assert(expectBuild(c) != nil).IsTrue()
		})

		g.It("Should fail when no image", func() {
			c := &common.Config{
				Build: &common.Step{},
			}
			g.Assert(expectImage(c) != nil).IsTrue()
		})

		g.It("Should fail when no commands", func() {
			c := &common.Config{
				Build: &common.Step{},
			}
			g.Assert(expectCommand(c) != nil).IsTrue()
		})

		g.It("Should pass when proper Build provided", func() {
			c := &common.Config{
				Build: &common.Step{
					Config: map[string]interface{}{
						"commands": []string{"echo hi"},
					},
				},
			}
			g.Assert(expectImage(c) != nil).IsTrue()
		})

		g.It("Should fail when untrusted setup image", func() {
			c := &common.Config{Setup: &common.Step{Image: "foo/bar"}}
			g.Assert(expectTrustedSetup(c) != nil).IsTrue()
		})

		g.It("Should fail when untrusted clone image", func() {
			c := &common.Config{Clone: &common.Step{Image: "foo/bar"}}
			g.Assert(expectTrustedClone(c) != nil).IsTrue()
		})

		g.It("Should fail when untrusted publish image", func() {
			c := &common.Config{}
			c.Publish = map[string]*common.Step{}
			c.Publish["docker"] = &common.Step{Image: "foo/bar"}
			g.Assert(expectTrustedPublish(c) != nil).IsTrue()
		})

		g.It("Should fail when untrusted deploy image", func() {
			c := &common.Config{}
			c.Deploy = map[string]*common.Step{}
			c.Deploy["amazon"] = &common.Step{Image: "foo/bar"}
			g.Assert(expectTrustedDeploy(c) != nil).IsTrue()
		})

		g.It("Should fail when untrusted notify image", func() {
			c := &common.Config{}
			c.Notify = map[string]*common.Step{}
			c.Notify["hipchat"] = &common.Step{Image: "foo/bar"}
			g.Assert(expectTrustedNotify(c) != nil).IsTrue()
		})

		g.It("Should pass linter when build properly setup", func() {
			c := &common.Config{}
			c.Build = &common.Step{}
			c.Build.Image = "golang"
			c.Build.Config = map[string]interface{}{}
			c.Build.Config["commands"] = []string{"go build", "go test"}
			c.Publish = map[string]*common.Step{}
			c.Publish["docker"] = &common.Step{Image: "docker"}
			c.Deploy = map[string]*common.Step{}
			c.Deploy["kubernetes"] = &common.Step{Image: "kubernetes"}
			c.Notify = map[string]*common.Step{}
			c.Notify["email"] = &common.Step{Image: "email"}
			g.Assert(Lint(c) == nil).IsTrue()
		})

	})
}

func Test_LintPlugins(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Plugin Linter", func() {

		g.It("Should fail un-trusted plugin", func() {
			c := &common.Config{
				Setup:   &common.Step{Image: "foo/baz"},
				Clone:   &common.Step{Image: "foo/bar"},
				Notify:  map[string]*common.Step{},
				Deploy:  map[string]*common.Step{},
				Publish: map[string]*common.Step{},
			}
			o := &Opts{Whitelist: []string{"plugins/*"}}
			g.Assert(LintPlugins(c, o) != nil).IsTrue()
		})

		g.It("Should pass when empty whitelist", func() {
			c := &common.Config{
				Setup:   &common.Step{Image: "foo/baz"},
				Clone:   &common.Step{Image: "foo/bar"},
				Notify:  map[string]*common.Step{},
				Deploy:  map[string]*common.Step{},
				Publish: map[string]*common.Step{},
			}
			o := &Opts{Whitelist: []string{}}
			g.Assert(LintPlugins(c, o) == nil).IsTrue()
		})

		g.It("Should pass wildcard", func() {
			c := &common.Config{
				Setup:   &common.Step{Image: "plugins/drone-setup"},
				Clone:   &common.Step{Image: "plugins/drone-build"},
				Notify:  map[string]*common.Step{},
				Deploy:  map[string]*common.Step{},
				Publish: map[string]*common.Step{},
			}
			o := &Opts{Whitelist: []string{"plugins/*"}}
			g.Assert(LintPlugins(c, o) == nil).IsTrue()
		})

		g.It("Should pass itemized", func() {
			c := &common.Config{
				Setup:   &common.Step{Image: "plugins/drone-setup"},
				Clone:   &common.Step{Image: "plugins/drone-build"},
				Notify:  map[string]*common.Step{},
				Deploy:  map[string]*common.Step{},
				Publish: map[string]*common.Step{},
			}
			o := &Opts{Whitelist: []string{"plugins/drone-setup", "plugins/drone-build"}}
			g.Assert(LintPlugins(c, o) == nil).IsTrue()
		})
	})
}
