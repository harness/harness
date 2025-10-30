#!/bin/bash
# Copyright 2022 Harness Inc. All rights reserved.
# Use of this source code is governed by the PolyForm Shield 1.0.0 license
# that can be found in the licenses directory at the root of this repository, also available at
# https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.

# Produces source files intended for use with add_license_header.sh
# This is the "Polyform only" variation.
# It produces 2 files, each file lists the source files which should be stamped with the associated header.
#   sources-free-trial.txt  ->  Polyform Free Trial
#   sources-shield.txt      ->  Polyform Shield
#

BASEDIR=$(dirname $0)

POLYFORM_FREE_TRIAL_FILE="$BASEDIR/.license-files-polyform-free-trial.txt"
SUPPORTED_EXTENSIONS_FILE="$BASEDIR/.license-extensions.txt"

FILE_EXTENSIONS=$(awk 'NR>1 {print $1}' "$SUPPORTED_EXTENSIONS_FILE" | paste -s -d '|' -)

###############################
## Validate file lists       ##
###############################
function print_possible_alternates {
  POTENTIAL_ALTERNATE=$(find . -name "$(basename "$1")")
  if [ ! -z "$POTENTIAL_ALTERNATE" ]; then
    echo "Has the file been moved to         $POTENTIAL_ALTERNATE"
  fi
}

function validate_missing {
  MISSING_FILES=$(sort -f "$1" | uniq -i | xargs ls 2>&1 | grep 'No such file')
  if [ ! -z "$MISSING_FILES" ]; then
    while read -r LINE; do
      FILE=$(echo "$LINE" | awk 'BEGIN{FS=": "}{print $2}')
      echo "ERROR: Skipping file as it does not exist $FILE"
      print_possible_alternates "$FILE"
      echo
    done <<< "$MISSING_FILES"
    echo ""
    echo "$1 contains non-existent files."
    exit 2
  fi
}

validate_missing "$POLYFORM_FREE_TRIAL_FILE"

###############################
## Polyform Free Trial       ##
###############################

function find_within_dir {
  DIRECTORIES=$(awk '{gsub(/\/[^\/]*$/, ""); print}' "$1" | sort -f | uniq -i)
  LAST_DIR=""
  test -e sources-all-within-dir.txt && rm sources-all-within-dir.txt
  while read -e DIR; do
    if [ $(grep -m 1 -cE "^$LAST_DIR/" <<< "$DIR") -eq 0 ]; then
      find "$DIR" -type f | grep -E "\.($FILE_EXTENSIONS)$" >> sources-all-within-dir.txt
      LAST_DIR="$DIR"
    fi
  done <<< "$DIRECTORIES"

  sort -f sources-all-within-dir.txt | uniq -i
}

# find all Polyform Free Trial files, including all files which reside within the same directory
find_within_dir "$POLYFORM_FREE_TRIAL_FILE" | sort -f | uniq -i > sources-free-trial.txt

rm sources-all-within-dir.txt

###############################
## Polyform Shield           ##
###############################

# find all files which should receive header
find . -type f -not -path "*/node_modules/*" | grep -E "\.($FILE_EXTENSIONS)$" | cut -c 3- | sort -f | uniq -i > sources-all.txt

# remove free trial files
comm -23i sources-all.txt sources-free-trial.txt | sort -f | uniq -i > sources-shield.txt

###############################
## Compute Counts            ##
###############################

FILE_COUNTS=$(wc -l sources-free-trial.txt sources-shield.txt)

# sanity check file counts
FILE_COUNT_SUM_OF_EACH_LICENSE=$(grep total <<< "$FILE_COUNTS" | awk '{print $1}')
FILE_COUNT=$(wc -l sources-all.txt | awk '{print $1}')
if [ "$FILE_COUNT_SUM_OF_EACH_LICENSE" -ne "$FILE_COUNT" ]; then
  echo "ERROR: File counts do not match!"
  echo "Sum of separate licenses: $FILE_COUNT_SUM_OF_EACH_LICENSE"
  echo "All licensed files: $FILE_COUNT"
  exit 1
fi

# print file and line counts
echo "File Count:"
echo "$FILE_COUNTS"
echo
echo "Line Count:"
LINE_COUNT_FREE_TRIAL=$(cat sources-free-trial.txt | tr \\n \\0 | xargs -0 wc -l | grep total | awk '{print $1}' | paste -s -d '+' - | bc)
LINE_COUNT_SHIELD=$(cat sources-shield.txt | tr \\n \\0 | xargs -0 wc -l | grep total | awk '{print $1}' | paste -s -d '+' - | bc)
printf "%8d Polyform Free Trial\n" "${LINE_COUNT_FREE_TRIAL}"
printf "%8d Polyform Shield\n" "${LINE_COUNT_SHIELD}"
printf "%8d total\n" "$(echo $LINE_COUNT_FREE_TRIAL+$LINE_COUNT_SHIELD | bc)"

#rm sources-all.txt sources-free-trial.txt sources-shield.txt
