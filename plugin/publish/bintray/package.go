package bintray

import (
	"fmt"
	"strings"

	"github.com/drone/drone/shared/build/buildfile"
)

const bintray_endpoint = "https://api.bintray.com/content/%s/%s/%s/%s/%s"

type Package struct {
	File       string   `yaml:"file"`
	Type       string   `yaml:"type"`
	Owner      string   `yaml:"owner"`
	Repository string   `yaml:"repository"`
	Package    string   `yaml:"package"`
	Version    string   `yaml:"version"`
	Target     string   `yaml:"target"`
	Distr      string   `yaml:"distr,omitempty"`
	Component  string   `yaml:"component,omitempty"`
	Arch       []string `yaml:"arch,omitempty"`
	Publish    bool     `yaml:"publish,omitempty"`
	Override   bool     `yaml:"override,omitempty"`
}

func (p *Package) Write(username, api_key string, f *buildfile.Buildfile) {
	if len(p.File) == 0 || len(p.Owner) == 0 || len(p.Repository) == 0 || len(p.Package) == 0 || len(p.Version) == 0 || len(p.Target) == 0 {
		f.WriteCmdSilent(`echo -e "Bintray Plugin: Missing argument(s)\n\n"`)

		if len(p.Package) == 0 {
			f.WriteCmdSilent(fmt.Sprintf(`echo -e "\tpackage not defined in yaml config"`))
			return
		}

		if len(p.File) == 0 {
			f.WriteCmdSilent(fmt.Sprintf(`echo -e "\tpackage %s: file not defined in yaml config"`, p.Package))
		}

		if len(p.Owner) == 0 {
			f.WriteCmdSilent(fmt.Sprintf(`echo -e "\tpackage %s: owner not defined in yaml config"`, p.Package))
		}

		if len(p.Repository) == 0 {
			f.WriteCmdSilent(fmt.Sprintf(`echo -e "\tpackage %s: repository not defined in yaml config"`, p.Package))
		}

		if len(p.Version) == 0 {
			f.WriteCmdSilent(fmt.Sprintf(`echo -e "\tpackage %s: version not defined in yaml config"`, p.Package))
		}

		if len(p.Target) == 0 {
			f.WriteCmdSilent(fmt.Sprintf(`echo -e "\tpackage %s: target not defined in yaml config"`, p.Package))
		}

		f.WriteCmdSilent("exit 1")

		return
	}

	switch p.Type {
	case "deb":
		p.debUpload(username, api_key, f)
	case "rpm":
		p.upload(username, api_key, f)
	case "maven":
		p.upload(username, api_key, f)
	default:
		p.upload(username, api_key, f)
	}
}

func (p *Package) debUpload(username, api_key string, f *buildfile.Buildfile) {
	if len(p.Distr) == 0 || len(p.Component) == 0 || len(p.Arch) == 0 {
		f.WriteCmdSilent(`echo -e "Bintray Plugin: Missing argument(s)\n\n"`)

		if len(p.Distr) == 0 {
			f.WriteCmdSilent(fmt.Sprintf(`echo -e "\tpackage %s: distr not defined in yaml config"`, p.Package))
		}

		if len(p.Component) == 0 {
			f.WriteCmdSilent(fmt.Sprintf(`echo -e "\tpackage %s: component not defined in yaml config"`, p.Package))
		}

		if len(p.Arch) == 0 {
			f.WriteCmdSilent(fmt.Sprintf(`echo -e "\tpackage %s: arch not defined in yaml config"`, p.Package))
		}

		f.WriteCmdSilent("exit 1")

		return
	}

	f.WriteCmdSilent(fmt.Sprintf(`echo -e "\nUpload %s to %s/%s/%s"`, p.File, p.Owner, p.Repository, p.Package))
	f.WriteCmdSilent(fmt.Sprintf("curl -s -T %s -u%s:%s %s\\;deb_distribution\\=%s\\;deb_component\\=%s\\;deb_architecture=\\%s\\;publish\\=%d\\;override\\=%d",
		p.File, username, api_key, p.getEndpoint(), p.Distr, p.Component, strings.Join(p.Arch, ","), boolToInt(p.Publish), boolToInt(p.Override)))

}

func (p *Package) upload(username, api_key string, f *buildfile.Buildfile) {
	f.WriteCmdSilent(fmt.Sprintf(`echo -e "\nUpload %s to %s/%s/%s"`, p.File, p.Owner, p.Repository, p.Package))
	f.WriteCmdSilent(fmt.Sprintf("curl -s -T %s -u%s:%s %s\\;publish\\=%d\\;override\\=%d",
		p.File, username, api_key, p.getEndpoint(), boolToInt(p.Publish), boolToInt(p.Override)))
}

func (p *Package) getEndpoint() string {
	return fmt.Sprintf(bintray_endpoint, p.Owner, p.Repository, p.Package, p.Version, p.Target)
}

func boolToInt(val bool) int {
	if val {
		return 1
	} else {
		return 0
	}
}
