#!/bin/sh
PROFILE_FILE="/etc/profile"
echo "Processing variables:"
# Environment variables to process
env_variables="
{{- range .EnvVariables }}
{{ . }}
{{- end }}
"
# Process each variable in the env_variables array
echo "$env_variables" | while IFS= read -r line; do  # Skip empty lines
  [ -z "$line" ] && continue

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