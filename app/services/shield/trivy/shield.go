// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package trivy

import (
	"crypto/sha256"
	"encoding/hex"
	"math"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/harness/gitness/app/services/shield/sast"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ScannerTrivy identifies Trivy results in types.ShieldFinding.Scanner.
const ScannerTrivy = enum.ShieldScannerTrivy

// Ecosystem values of image findings.
const (
	EcosystemOSPackage = "os-pkg"
	EcosystemLibrary   = "library"
)

// ImageFinding is a normalized vulnerability found in a container image.
type ImageFinding struct {
	// VulnID is the SARIF rule id, usually a CVE id (may also be a GHSA or distro advisory id).
	VulnID string
	// CVE is the CVE id named in the message when it differs from VulnID.
	CVE string
	// Ecosystem is "os-pkg" or "library".
	Ecosystem        string
	Package          string
	InstalledVersion string
	// FixedVersions is empty when no fix is available.
	FixedVersions []string
	Severity      sast.Severity
	// Score is the CVSS-like score in [0, 10] (from security-severity, else derived from Severity).
	Score float64
	Title string
	// Target is the scanned artifact the package belongs to (image name or lock file path).
	Target      string
	Fingerprint string
}

// ConfigFinding is a normalized IaC misconfiguration.
type ConfigFinding struct {
	// RuleID is the check id, e.g. "AVD-AWS-0086".
	RuleID      string
	Title       string
	Message     string
	Severity    sast.Severity
	Score       float64
	Path        string
	StartLine   int
	EndLine     int
	Fingerprint string
}

var (
	pkgRe       = regexp.MustCompile(`(?m)^Package:[ \t]*(.+?)[ \t]*$`)
	installedRe = regexp.MustCompile(`(?m)^Installed Version:[ \t]*(.+?)[ \t]*$`)
	fixedRe     = regexp.MustCompile(`(?m)^Fixed Version:[ \t]*(.+?)[ \t]*$`)
	vulnRe      = regexp.MustCompile(`(?m)^Vulnerability[: \t]+(\S+)[ \t]*$`)
	severityRe  = regexp.MustCompile(`(?m)^Severity:[ \t]*(\w+)`)
	messageRe   = regexp.MustCompile(`(?m)^Message:[ \t]*(.+?)[ \t]*$`)
	cveRe       = regexp.MustCompile(`(?i)^CVE-\d{4}-\d+$`)
)

// libraryTargetSuffixes are file names/extensions that mark a language-package target
// (lock files, manifests, archives) as opposed to an OS package target (the image name).
var libraryTargetSuffixes = []string{
	".json", ".lock", ".txt", ".mod", ".sum", ".toml", ".xml", ".gradle", ".yaml", ".yml",
	".jar", ".war", ".ear", ".par", ".gemspec", ".whl", ".egg", ".nupkg", ".csproj", ".config",
	".deps", ".sbt", "/metadata", "/pkg-info", "/gemfile", "/pipfile",
}

// ParseImageFindings maps the results of a `trivy image` SARIF report to image findings.
// Results without a rule id or package are skipped.
func ParseImageFindings(rep *sast.Report) []ImageFinding {
	var out []ImageFinding
	forEachResult(rep, func(res *sast.Result, rule *sast.Rule, ruleID string) {
		msg := res.Message.Text
		pkg := firstGroup(pkgRe, msg)
		if pkg == "" {
			return
		}
		f := ImageFinding{
			VulnID:           ruleID,
			Package:          pkg,
			InstalledVersion: firstGroup(installedRe, msg),
			FixedVersions:    splitVersions(firstGroup(fixedRe, msg)),
			Title:            ruleTitle(rule, ruleID),
		}
		if id := firstGroup(vulnRe, msg); id != "" && id != ruleID && cveRe.MatchString(id) {
			f.CVE = id
		}
		f.Severity, f.Score = severityOf(firstGroup(severityRe, msg), res, rule)
		if len(res.Locations) > 0 {
			f.Target = normalizePath(res.Locations[0].PhysicalLocation.ArtifactLocation.URI)
		}
		f.Ecosystem = ecosystemOf(f.Target)
		f.Fingerprint = strings.Join([]string{"image", ruleID, pkg, f.InstalledVersion, f.Target}, "|")
		out = append(out, f)
	})
	return out
}

// ParseConfigFindings maps the results of a `trivy config` SARIF report to IaC findings.
// Results without a rule id are skipped.
func ParseConfigFindings(rep *sast.Report) []ConfigFinding {
	var out []ConfigFinding
	forEachResult(rep, func(res *sast.Result, rule *sast.Rule, ruleID string) {
		msg := res.Message.Text
		f := ConfigFinding{RuleID: ruleID, Title: ruleTitle(rule, ruleID)}
		f.Message = firstNonEmpty(firstGroup(messageRe, msg), msg)
		f.Severity, f.Score = severityOf(firstGroup(severityRe, msg), res, rule)
		var snippet string
		if len(res.Locations) > 0 {
			pl := res.Locations[0].PhysicalLocation
			f.Path = normalizePath(pl.ArtifactLocation.URI)
			f.StartLine = pl.Region.StartLine
			f.EndLine = max(pl.Region.EndLine, pl.Region.StartLine)
			snippet = pl.Region.Snippet.Text
		}
		// Line numbers are excluded so that moving a resource keeps the identity.
		disc := strings.Join(strings.Fields(snippet), " ")
		if disc == "" {
			disc = strings.Join(strings.Fields(f.Message), " ")
		}
		sum := sha256.Sum256([]byte(disc))
		f.Fingerprint = "iac|" + ruleID + "|" + f.Path + "|" + hex.EncodeToString(sum[:8])
		out = append(out, f)
	})
	return out
}

// ToShieldImageFinding maps an image finding to a new open Shield finding (kind image) observed
// in scanID. The image reference or lock file path is stored in ManifestPath.
func ToShieldImageFinding(f ImageFinding, repoID, scanID, now int64) *types.ShieldFinding {
	aliases := []string{}
	if f.CVE != "" {
		aliases = append(aliases, f.CVE)
	}
	fixed := f.FixedVersions
	if fixed == nil {
		fixed = []string{}
	}
	sf := newShieldFinding(repoID, scanID, now, f.Fingerprint, enum.ShieldFindingKindImage, f.Score)
	sf.Ecosystem = f.Ecosystem
	sf.Package = f.Package
	sf.InstalledVersion = f.InstalledVersion
	sf.ManifestPath = f.Target
	sf.VulnID = f.VulnID
	sf.Aliases = aliases
	sf.Summary = f.Title
	sf.FixedVersions = fixed
	if len(fixed) > 0 {
		sf.MinFixedVersion = fixed[0]
	}
	return sf
}

// ToShieldConfigFinding maps an IaC finding to a new open Shield finding (kind iac) observed in
// scanID. The file path is stored in ManifestPath; line numbers are not persisted.
func ToShieldConfigFinding(f ConfigFinding, repoID, scanID, now int64) *types.ShieldFinding {
	sf := newShieldFinding(repoID, scanID, now, f.Fingerprint, enum.ShieldFindingKindIaC, f.Score)
	sf.ManifestPath = f.Path
	sf.VulnID = f.RuleID
	sf.Summary = firstNonEmpty(f.Message, f.Title)
	return sf
}

func newShieldFinding(
	repoID, scanID, now int64,
	fingerprint string,
	kind enum.ShieldFindingKind,
	score float64,
) *types.ShieldFinding {
	return &types.ShieldFinding{
		RepoID:          repoID,
		Fingerprint:     fingerprint,
		Scanner:         ScannerTrivy,
		Kind:            kind,
		Aliases:         []string{},
		CVSSScore:       score,
		Reachable:       enum.ShieldReachabilityUnknown,
		RiskScore:       max(0, min(100, int(math.Round(score*10)))),
		FixedVersions:   []string{},
		State:           enum.ShieldFindingStateOpen,
		FirstSeenScanID: scanID,
		LastSeenScanID:  scanID,
		FirstSeen:       now,
		LastSeen:        now,
		Created:         now,
		Updated:         now,
	}
}

func forEachResult(rep *sast.Report, fn func(res *sast.Result, rule *sast.Rule, ruleID string)) {
	if rep == nil {
		return
	}
	for _, run := range rep.Runs {
		rules := run.Tool.Driver.Rules
		byID := make(map[string]*sast.Rule, len(rules))
		for i := range rules {
			byID[rules[i].ID] = &rules[i]
		}
		for i := range run.Results {
			res := &run.Results[i]
			rule := byID[res.RuleID]
			if rule == nil && res.RuleIndex != nil && *res.RuleIndex >= 0 && *res.RuleIndex < len(rules) {
				rule = &rules[*res.RuleIndex]
			}
			ruleID := res.RuleID
			if ruleID == "" && rule != nil {
				ruleID = rule.ID
			}
			if ruleID == "" {
				continue
			}
			fn(res, rule, ruleID)
		}
	}
}

func ruleTitle(rule *sast.Rule, ruleID string) string {
	if rule == nil {
		return ruleID
	}
	return firstNonEmpty(rule.ShortDescription.Text, rule.FullDescription.Text, rule.Name, ruleID)
}

// severityOf prefers the severity named in the message, then the rule tags, then the
// security-severity score and finally the SARIF level.
func severityOf(msgSev string, res *sast.Result, rule *sast.Rule) (sast.Severity, float64) {
	sev := parseSeverity(msgSev)
	if sev == "" {
		tags := append([]string{}, res.Properties.Tags...)
		if rule != nil {
			tags = append(tags, rule.Properties.Tags...)
		}
		for _, t := range tags {
			if sev = parseSeverity(t); sev != "" {
				break
			}
		}
	}
	score := res.Properties.SecuritySeverity
	if score == nil && rule != nil {
		score = rule.Properties.SecuritySeverity
	}
	if score != nil {
		s := math.Max(0, math.Min(10, *score))
		if sev == "" {
			sev = bucket(s)
		}
		return sev, s
	}
	if sev == "" {
		switch strings.ToLower(res.Level) {
		case "error":
			sev = sast.SeverityHigh
		case "warning":
			sev = sast.SeverityMedium
		default:
			sev = sast.SeverityLow
		}
	}
	return sev, defaultScore(sev)
}

func parseSeverity(s string) sast.Severity {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "CRITICAL":
		return sast.SeverityCritical
	case "HIGH":
		return sast.SeverityHigh
	case "MEDIUM", "MODERATE":
		return sast.SeverityMedium
	case "LOW", "UNKNOWN":
		return sast.SeverityLow
	}
	return ""
}

func bucket(score float64) sast.Severity {
	switch {
	case score >= 9:
		return sast.SeverityCritical
	case score >= 7:
		return sast.SeverityHigh
	case score >= 4:
		return sast.SeverityMedium
	default:
		return sast.SeverityLow
	}
}

func defaultScore(s sast.Severity) float64 {
	switch s {
	case sast.SeverityCritical:
		return 9.5
	case sast.SeverityHigh:
		return 8.0
	case sast.SeverityMedium:
		return 5.0
	default:
		return 2.0
	}
}

// ecosystemOf classifies the scan target: lock files and manifests hold library packages,
// anything else (the image name / OS release) holds OS packages.
func ecosystemOf(target string) string {
	t := strings.ToLower(target)
	for _, s := range libraryTargetSuffixes {
		if strings.HasSuffix(t, s) {
			return EcosystemLibrary
		}
	}
	switch path.Base(t) {
	case "gemfile", "pipfile":
		return EcosystemLibrary
	}
	return EcosystemOSPackage
}

func splitVersions(s string) []string {
	var out []string
	for _, v := range strings.Split(s, ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func firstGroup(re *regexp.Regexp, s string) string {
	if m := re.FindStringSubmatch(s); m != nil {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func normalizePath(uri string) string {
	if u, err := url.Parse(uri); err == nil && u.Path != "" && (u.Scheme == "" || u.Scheme == "file") {
		uri = u.Path
	}
	uri = strings.TrimPrefix(uri, "./")
	return strings.TrimPrefix(uri, "/")
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
