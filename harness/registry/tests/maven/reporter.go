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

package mavenconformance

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/onsi/ginkgo/v2/types"
)

// Status constants for test results.
const (
	StatusPassed  = "passed"
	StatusFailed  = "failed"
	StatusSkipped = "skipped"
	StatusPending = "pending"
)

// TestResult represents a single test result.
type TestResult struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Error     string    `json:"error,omitempty"`
	Output    string    `json:"output,omitempty"`
}

// TestReport represents the overall test report.
type TestReport struct {
	StartTime   time.Time    `json:"start_time"`
	EndTime     time.Time    `json:"end_time"`
	TestResults []TestResult `json:"test_results"`
	Summary     struct {
		Passed  int `json:"passed"`
		Failed  int `json:"failed"`
		Pending int `json:"pending"`
		Skipped int `json:"skipped"`
		Total   int `json:"total"`
	} `json:"summary"`
}

var (
	report = TestReport{
		StartTime:   time.Now(),
		TestResults: make([]TestResult, 0),
	}
)

// ReportTest adds a test result to the report.
func ReportTest(name string, status string, err error) {
	// Create detailed output with available information.
	detailedOutput := fmt.Sprintf("Test: %s\nStatus: %s\n", name, status)
	detailedOutput += fmt.Sprintf("Time: %s\n", time.Now().Format(time.RFC3339))

	// Add more context based on test status.
	switch status {
	case StatusPassed:
		detailedOutput += "Result: Test completed successfully\n"
	case StatusFailed:
		detailedOutput += "Result: Test failed with errors\n"
	case StatusSkipped:
		detailedOutput += "Result: Test was skipped\n"
	case StatusPending:
		detailedOutput += "Result: Test is pending implementation\n"
	}

	// Create a new test result with detailed information.
	result := TestResult{
		Name:      name,
		Status:    status,
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Output:    detailedOutput,
	}

	// Capture error information if available.
	if err != nil {
		result.Error = err.Error()
		result.Output += fmt.Sprintf("Error: %v\n", err)
		// Add stack trace or additional context if available.
		log.Printf("Test failed: %s - %v", name, err)
	}

	// Add the result to the report.
	report.TestResults = append(report.TestResults, result)
}

// SaveReport saves the test report to a file.
// If logSummary is true, a summary of test results will be logged to stdout.
func SaveReport(filename string, logSummary bool) error {
	// Update summary statistics.
	report.EndTime = time.Now()
	report.Summary.Passed = 0
	report.Summary.Failed = 0
	report.Summary.Skipped = 0
	report.Summary.Pending = 0
	report.Summary.Total = len(report.TestResults)

	// Count test results by status.
	for _, result := range report.TestResults {
		switch result.Status {
		case StatusPassed:
			report.Summary.Passed++
		case StatusFailed:
			report.Summary.Failed++
		case StatusSkipped:
			report.Summary.Skipped++
		case StatusPending:
			report.Summary.Pending++
		}
	}

	// Generate JSON report.
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	// Write the JSON report to file.
	if err := os.WriteFile(filename, data, 0600); err != nil {
		return err
	}

	// Log the summary only if requested.
	if logSummary {
		log.Printf("Maven test results: %d passed, %d failed, %d pending, %d skipped (total: %d)",
			report.Summary.Passed, report.Summary.Failed, report.Summary.Pending,
			report.Summary.Skipped, report.Summary.Total)
	}

	return nil
}

// ReporterHook implements ginkgo's Reporter interface.
type ReporterHook struct{}

func (r *ReporterHook) SuiteWillBegin() {
	report.StartTime = time.Now()
}

func (r *ReporterHook) SuiteDidEnd() {
	report.EndTime = time.Now()
	// Pass true to log summary in the ReporterHook context.
	if err := SaveReport("maven_conformance_report.json", true); err != nil {
		log.Printf("Failed to save report: %v\n", err)
	}
}

func (r *ReporterHook) SpecDidComplete(text string, state types.SpecState, failure error) {
	// Determine test status.
	var status string
	switch state {
	case types.SpecStateSkipped:
		status = StatusSkipped
	case types.SpecStateFailed:
		status = StatusFailed
	case types.SpecStatePending:
		status = StatusPending
	case types.SpecStatePassed:
		status = StatusPassed
	case types.SpecStateInvalid, types.SpecStateAborted, types.SpecStatePanicked,
		types.SpecStateInterrupted, types.SpecStateTimedout:
		status = StatusFailed
	}

	// Log detailed test information.
	log.Printf("Test completed: %s - %s", text, status)

	// Report the test with detailed information.
	ReportTest(text, status, failure)
}
