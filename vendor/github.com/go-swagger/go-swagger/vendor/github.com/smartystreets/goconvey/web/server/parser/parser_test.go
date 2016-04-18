package parser

import (
	"errors"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/smartystreets/goconvey/web/server/contract"
)

func TestParser(t *testing.T) {

	Convey("Subject: Parser parses test output for active packages", t, func() {
		packages := []*contract.Package{
			&contract.Package{Ignored: false, Output: "Active", Result: contract.NewPackageResult("asdf")},
			&contract.Package{Ignored: true, Output: "Inactive", Result: contract.NewPackageResult("qwer")},
		}
		parser := NewParser(fakeParserImplementation)

		Convey("When given a collection of packages", func() {
			parser.Parse(packages)

			Convey("The parser uses its internal parsing mechanism to parse the output of only the active packages", func() {
				So(packages[0].Result.Outcome, ShouldEqual, packages[0].Output)
			})

			Convey("The parser should mark inactive packages as ignored", func() {
				So(packages[1].Result.Outcome, ShouldEqual, contract.Ignored)
			})
		})

		Convey("When a package could not be tested (maybe it was deleted between scanning and execution?)", func() {
			packages[0].Output = ""
			packages[0].Error = errors.New("Directory does not exist")

			parser.Parse(packages)

			Convey("The package result should not be parsed and the outcome should actually resemble the problem", func() {
				So(packages[0].Result.Outcome, ShouldEqual, contract.TestRunAbortedUnexpectedly)
			})
		})
	})
}

func fakeParserImplementation(result *contract.PackageResult, rawOutput string) {
	result.Outcome = rawOutput
}
