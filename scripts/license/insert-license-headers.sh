#!/usr/bin/env bash
# Copyright 2023 Harness, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

function usage {
  echo "Script to add license header to files"
  echo
  echo "./$(basename $0) -l <file> [-s <file>] [-f <file>]"
  echo "Options"
  echo "  -h        prints usage"
  echo "  -l <file> path to file containing license text"
  echo "  -s <file> path to file containing source files to process"
  echo "  -f <file> path to a single file to process"
}

function error {
  usage
  echo
  echo "$1"
}

function error_missing_file {
  error "ERROR: $1 file is required!"
  exit 3
}

function error_cannot_read_file {
  error "ERROR: Cannot open file for reading $1"
  exit 4
}

while getopts ":df:hl:s:v" arg; do
  case ${arg} in
    f) SOURCE_FILES=${OPTARG} ;;
    h) usage ; exit 0 ;;
    l) PATH_TO_LICENSE=${OPTARG} ;;
    s) PATH_TO_INPUT=${OPTARG} ;;
    v) VERBOSE="ON" ;;
    :)
       echo "$0: Must supply an argument to -$OPTARG." >&2
       exit 1 ;;
    ?)
       echo "Invalid option: -${OPTARG}."
       exit 2 ;;
  esac
done

if [ -z $PATH_TO_LICENSE ]; then
  error_missing_file "License"
fi

if [ -z "$SOURCE_FILES" -a -z "$PATH_TO_INPUT" ]; then
  error_missing_file "Source"
fi

if [ ! -r $PATH_TO_LICENSE ]; then
  error_cannot_read_file "$PATH_TO_LICENSE"
fi

if [ -z "$SOURCE_FILES" -a ! -r "$PATH_TO_INPUT" ]; then
  error_cannot_read_file "$PATH_TO_INPUT"
fi

############################
##  Common Functions      ##
############################

function debug {
  if [ "$VERBOSE" = "ON" ]; then
    echo $1
  fi
}

function create_output_file {
  FILE_COUNT=$(( FILE_COUNT + 1 ))
  LINE_COUNT=$(( LINE_COUNT + $(wc -l "$FILE" | awk '{print $1}') ))
  cp -p "$FILE" "$NEW_FILE"
  : > "$NEW_FILE"
}

function add_header_if_required {
  HEADER_WITHOUT_YEAR=$(sed 's/ 20[0-9][0-9] / <YEAR> /' <<< "$HEADER_WITHOUT_COMMENT_SYMBOL")
  if [ "$HEADER_WITHOUT_YEAR" = "$LICENSE_TEXT" ]; then
    debug "File has correct license header: $FILE"
  elif [ $(grep -m1 -ciE "(copyright|license)" <<<"$EXISTING_HEADER") -eq 1 ]; then
    debug "Skipping file with alternate license header: $FILE"
  else
    $1
  fi
}

function write_file_header {
  FILE_DATE=$(git log -1 --format="%ad" --date=format:%Y -- "$FILE")
  FILE_DATE=${FILE_DATE:-$(date +'%Y')}
  while IFS='' read -r license_line; do 
    if [ -z "$license_line" ]; then
      echo "$SYMBOL" >> "$NEW_FILE"
    else
      echo "$SYMBOL $license_line" | sed "s/<YEAR>/${FILE_DATE}/" >> "$NEW_FILE"
    fi
  done <"$PATH_TO_LICENSE"
}

function write_remaining_file_content {
  if [ ! -z "$(head -1 <<<"$FILE_CONTENT")" ]; then
    echo >> "$NEW_FILE"
  fi
  echo "$FILE_CONTENT" >> "$NEW_FILE"
  mv "$NEW_FILE" "$FILE"
}

############################
##  Double Slash          ##
############################

function handle_double_slash {
  EXISTING_HEADER=$(read_header_double_slash)
  if [ -z "$EXISTING_HEADER" ]; then
    write_file_double_slash
  else
    HEADER_WITHOUT_COMMENT_SYMBOL=$(cut -c 4- <<<"$EXISTING_HEADER")
    add_header_if_required "write_file_double_slash"
  fi
}

function read_header_double_slash {
  awk '{ if (/^\/\//) {print} else {exit} }' "$FILE"
}

function write_file_double_slash {
  debug "Adding license header: $FILE"
  NEW_FILE="$FILE.new"
  SYMBOL="//"
  create_output_file
  write_file_header
  write_remaining_file_content
}

############################
##  Slash Star            ##
############################

function handle_slash_star {
  EXISTING_HEADER=$(read_header_slash_star)
  if [ -z "$EXISTING_HEADER" ]; then
    write_file_slash_star
  else
    HEADER_WITHOUT_COMMENT_SYMBOL=$(cut -c 4- <<<"$EXISTING_HEADER" | awk 'NR > 1')
    add_header_if_required "write_file_slash_star"
  fi
}

function read_header_slash_star {
  awk '/\/\*/ {isHeader=1}; isHeader == 1 {print}; /\*\// {exit}' "$FILE"
}

function write_file_slash_star {
  debug "Adding license header: $FILE"
  NEW_FILE="$FILE.new"
  SYMBOL=" *"
  create_output_file
  echo "/*" > "$NEW_FILE"
  write_file_header
  echo " */" >> "$NEW_FILE"
  write_remaining_file_content
}

############################
##  Hash                  ##
############################

function handle_hash {
  RAW_HEADER=$(read_header_hash)
  EXISTING_HEADER=$(grep -v "^#!" <<<"$RAW_HEADER")
  IS_MISSING_SHE_BANG=$(test "$RAW_HEADER" = "$EXISTING_HEADER" && echo "TRUE")

  if [ -z "$EXISTING_HEADER" ]; then
    write_file_hash
  else
    HEADER_WITHOUT_COMMENT_SYMBOL=$(cut -c 3- <<<"$EXISTING_HEADER")
    add_header_if_required "write_file_hash"
  fi
}

function read_header_hash {
  awk '{ if (/^#/) {print} else {exit} }' "$FILE"
}

function write_file_hash {
  debug "Adding license header: $FILE"
  NEW_FILE="$FILE.new"
  SYMBOL="#"
  create_output_file
  if [ "$IS_MISSING_SHE_BANG" != "TRUE" ]; then
    head -1 <<<"$FILE_CONTENT" > "$NEW_FILE"
  fi
  write_file_header
  if [ "$IS_MISSING_SHE_BANG" != "TRUE" ]; then
    FILE_CONTENT=$(echo "$FILE_CONTENT" | awk "NR > 1")
  fi
  write_remaining_file_content
}

############################
##  Double Hyphen         ##
############################

function handle_double_hyphen {
  EXISTING_HEADER=$(read_header_double_hyphen)
  if [ -z "$EXISTING_HEADER" ]; then
    write_file_double_hyphen
  else
    HEADER_WITHOUT_COMMENT_SYMBOL=$(cut -c 4- <<<"$EXISTING_HEADER")
    add_header_if_required "write_file_double_hyphen"
  fi
}

function read_header_double_hyphen {
  awk '{ if (/^--/) {print} else {exit} }' "$FILE"
}

function write_file_double_hyphen {
  debug "Adding license header: $FILE"
  NEW_FILE="$FILE.new"
  SYMBOL="--"
  create_output_file
  write_file_header
  write_remaining_file_content
}

############################
##  Execution Functions   ##
############################

function handle_directory {
  FILE_EXTENSIONS=$(awk 'NR>1 {print $1}' "$SUPPORTED_EXTENSIONS_FILE" | paste -s -d '|' -)
  FILES_IN_DIR=$(find "$FILE" -type f | grep -E "\.($FILE_EXTENSIONS)$")
  while read -r FILE; do
    handle_file_based_on_extension
  done <<< "$FILES_IN_DIR"
}

function handle_file_based_on_extension {
  FILE_TYPE=$(basename "$FILE" | awk '{gsub(/.*\./, ""); print}' <<<"$FILE")
  FILE_CONTENT=$(cat "$FILE")
  FILE_TYPE_HANDLER=$(grep "$FILE_TYPE " "$SUPPORTED_EXTENSIONS_FILE" | awk '{print $3}')

  if [ -z "$FILE_TYPE_HANDLER" ]; then
    debug "Skipping file with extension '$FILE_TYPE' as it is not a supported filetype, file is $FILE"
  else
    handle_${FILE_TYPE_HANDLER}
  fi
}

############################
##  Execution             ##
############################

FILE_COUNT=0
LINE_COUNT=0
SCRIPT_DIR=$(dirname "$0")
SUPPORTED_EXTENSIONS_FILE="$SCRIPT_DIR/.license-extensions.txt"
LICENSE_TEXT=$(cat $PATH_TO_LICENSE)
if [ ! -z "$PATH_TO_INPUT" ]; then
  SOURCE_FILES=$(cat $PATH_TO_INPUT)
fi
PREVIOUSLY_OVERWRITTEN_HEADER=""

while read -r FILE; do
  if [ ! -e "$FILE" ]; then
    debug "Skipping file as it does not exist $FILE"
    continue
  elif [ -d "$FILE" ]; then
    handle_directory
    continue
  elif [ ! -w "$FILE" ]; then
    echo "ERROR: Skipping file as it is not writable $FILE"
    continue
  fi

  handle_file_based_on_extension
done <<< "$SOURCE_FILES"

if [ "$FILE_COUNT" -gt 0 ]; then
  echo "License added to $FILE_COUNT files with a total line count of $LINE_COUNT"
fi