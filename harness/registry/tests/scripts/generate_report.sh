outputFileName=$1
package=$2
EXIT_CODE=0

# Extract test statistics from output for display
passed=0
failed=0
skipped=0
total=0

if [ -f "$outputFileName" ]; then
  # Try to find success line first
  if grep -q "SUCCESS.*Passed.*Failed.*Pending.*Skipped" $outputFileName; then
      line=$(grep "SUCCESS.*Passed.*Failed.*Pending.*Skipped" $outputFileName | head -1)
  # If not found, try to find failure line
  elif grep -q "FAIL.*Passed.*Failed.*Pending.*Skipped" $outputFileName; then
      line=$(grep "FAIL.*Passed.*Failed.*Pending.*Skipped" $outputFileName | head -1)
      # Preserve the failure status
      EXIT_CODE=1
      echo "$package test failures detected"
  else
      # If no status line found, report that as an error
      line="ERROR: Could not determine test results"
      EXIT_CODE=1
      echo "Error: Could not find test result summary in output"
  fi
  
  # Extract test statistics from output
  passed=$(echo "$line" | grep -o "[0-9]\+ Passed" | awk '{print $1}')
  failed=$(echo "$line" | grep -o "[0-9]\+ Failed" | awk '{print $1}')
  pending=$(echo "$line" | grep -o "[0-9]\+ Pending" | awk '{print $1}')
  skipped=$(echo "$line" | grep -o "[0-9]\+ Skipped" | awk '{print $1}')
  
  # Set defaults if values are empty
  passed=${passed:-0}
  failed=${failed:-0}
  pending=${pending:-0}
  skipped=${skipped:-0}
  total=$((passed + failed + pending + skipped))
  
  # Display test results
  echo "$package test results: $passed passed, $failed failed, $pending pending, $skipped skipped (total: $total)"
  
  # Generate a JSON report from the test output
  echo "Generating $package conformance test report..."
  
  # Create a timestamp
  timestamp=$(date +"%Y-%m-%dT%H:%M:%S")
  
  # Create a JSON report
  json_report=$package"_conformance_report.json"
  cat > "$json_report" << EOL
{
  "timestamp": "$timestamp",
  "start_time": "$timestamp",
  "end_time": "$timestamp",
  "test_results": [
    {"name": "$package Registry Conformance Tests", "status": "$([ $EXIT_CODE -eq 0 ] && echo "passed" || echo "failed")", "start_time": "$timestamp", "end_time": "$timestamp"}
  ],
  "summary": {
    "passed": $passed,
    "failed": $failed,
    "pending": $pending,
    "skipped": $skipped,
    "total": $total
  }
}
EOL
fi
    
echo "$package report generated at $json_report"
    