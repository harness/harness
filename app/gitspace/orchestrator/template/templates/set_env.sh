#!/bin/sh
VARIABLES={{ .EnvVariables }}
# Check if the script is run as root, as modifying /etc/profile requires root privileges
if [[ $EUID -ne 0 ]]; then
  echo "This script must be run as root."
  exit 1
fi
# Path to /etc/profile
PROFILE_FILE="/etc/profile"
# Process each line in the VARIABLES string
echo "$VARIABLES" | while IFS= read -r line; do
  # Skip empty lines
  [[ -z "$line" ]] && continue

  # Extract the variable name and value
  var_name="${line%%=*}"  # Part before '='
  var_value="${line#*=}"  # Part after '='

  # Create the export statement
  export_statement="export $var_name=$var_value"

  # Check if the variable is already present in /etc/profile
  if ! grep -q "^export $var_name=" "$PROFILE_FILE"; then
    echo "$export_statement" >> "$PROFILE_FILE"
    echo "Added $export_statement to $PROFILE_FILE"
  else
    echo "$var_name is already present in $PROFILE_FILE"
  fi
done