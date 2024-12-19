#!/usr/bin/env bash

# Requirements:
# Run tests on the current and target branches. The directories are provided on the top.
# Find the total coverage of each of them.
# Success:
# 	current_branch_total_coverage >= target_branch_total_coverage
#	for the new files in the provided list of directories, the coverage should be atleast 75%   


set -euo pipefail

# This function needs some work!
calculate_file_coverage() {
    local file=$1
    local total_statements=0
    local covered_statements=0

    # Parse coverage report for the given file
    while read -r line; do
    	echo $line
        # Each line format: "<file>:<line>:<func_name>\t<coverage_percent>"
        # Extract statements and coverage percentage
        statements=$(echo "$line" | awk -F: '{print $4}' | awk '{print $2}')
        coverage=$(echo "$line" | awk -F: '{print $4}' | awk '{print $NF}' | sed 's/%//')

        # Accumulate total statements and covered statements
        total_statements=$((total_statements + statements))
        covered_statements=$(echo "$covered_statements + ($statements * $coverage / 100)" | bc)
    done < <(grep "$file" coverage_report.txt)

    # Calculate and print total coverage for the file
    if (( total_statements == 0 )); then
        echo "0.0"
    else
        echo "scale=2; ($covered_statements / $total_statements) * 100" | bc
    fi
}

# ============================
# Configuration Section
# ============================
# Provide the directories for which coverage should be checked.
# Directories can be nested. We'll add "/..." to each to ensure all sub-packages are included.
# Example: DIRECTORIES="./registry ./another/path"
DIRECTORIES="./registry"

if [ $# -lt 1 ]; then
    echo "Usage: $0 <target_branch>"
    exit 1
fi
TARGET_BRANCH=$1

# Convert directories into their recursive form (./dir => ./dir/...)
# to ensure nested packages are included
TEST_DIRS=""
for d in $DIRECTORIES; do
    # If it doesn't already end with "..." then append it
    if [[ "$d" != *"..." ]]; then
        d="${d%/}/..."
    fi
    TEST_DIRS="$TEST_DIRS $d"
done

# ============================
# Setup
# ============================
git fetch origin "${TARGET_BRANCH}:${TARGET_BRANCH}"
git checkout "${TARGET_BRANCH}"

# Run coverage on the target branch
go test -coverprofile=coverage_target.out $TEST_DIRS
TARGET_COV=$(go tool cover -func=coverage_target.out | grep total | awk '{print $3}' | sed 's/%//')

# Go back to the current (feature) branch
git checkout -

# Run coverage on the current (feature) branch
go test -coverprofile=coverage_current.out $TEST_DIRS
CURRENT_COV=$(go tool cover -func=coverage_current.out | grep total | awk '{print $3}' | sed 's/%//')

# Ensure the current branch coverage is not lower than the target branch coverage.
if (( $(echo "$CURRENT_COV < $TARGET_COV" | bc -l) )); then
    echo "Coverage decreased from ${TARGET_COV}% to ${CURRENT_COV}%."
    exit 1
fi
echo "Coverage checks passed! Current coverage: ${CURRENT_COV}%, Target coverage: ${TARGET_COV}%"

# Identify newly added files in the current branch compared to the target branch.
# NEW_FILES=$(git diff --name-status "${TARGET_BRANCH}...HEAD" | grep '^A' | awk '{print $2}' | grep '\.go$' | grep -v '_test\.go$')
# echo "new files: ""$NEW_FILES"

#go tool cover -func=coverage_current.out > coverage_report.txt
#
#for file in $NEW_FILES; do
#    # Check if the file is within one of the specified directories.
#    echo "new file is: "$file
#    IN_DIRS=false
#    for d in $DIRECTORIES; do
#        # Remove the trailing "/..." from the directory for a direct prefix check
#        base_dir="${d%/...}"
#        echo "base_dir value: "$d
#        normalized_base_dir="${base_dir#./}"
#        # If file starts with base_dir
#        if [[ $file == $normalized_base_dir* ]]; then
#            IN_DIRS=true
#            break
#        fi
#    done
#
#    if [ "$IN_DIRS" = true ]; then
#    	file_coverage=$(calculate_file_coverage "$file")
#        if (( $(echo "$file_coverage < 75.0" | bc -l) )); then
#            echo "New file $file has coverage of $file_coverage%, which is less than 75%."
#        else
#            echo "New file $file has coverage of $fcov%, which is better than 75%."
#        fi
#    fi
#done

echo "All checks passed! Current coverage: ${CURRENT_COV}%, Target coverage: ${TARGET_COV}%"
rm -rf coverage_*
