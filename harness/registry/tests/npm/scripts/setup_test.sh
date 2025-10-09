#!/bin/bash

# npm conformance test setup script
# This script handles authentication and test environment setup

# Helper function for logging
log() {
  echo "[NPM TEST] $1" >&2
}

# Use environment variables with defaults
SERVER_URL=${REGISTRY_SERVER_URL:-"localhost:3000"}  # Default to localhost:3000 if not provided
DEBUG=${REGISTRY_DEBUG:-"true"}                    # Default to debug mode enabled
TOKEN=${REGISTRY_TOKEN:-""}                       # Optional token for authentication
SPACE=${REGISTRY_SPACE:-""}                       # Optional space name
REGISTRY=${REGISTRY_NAME:-""}                     # Optional registry name

# Log the environment variables
log "Using environment variables:"
log "  SERVER_URL: $SERVER_URL"
log "  SPACE: $SPACE"
log "  REGISTRY: $REGISTRY"

# Set strict error handling
set -eo pipefail

# Setup test environment
setup_environment() {
  log "Setting up test environment..."
  log "Server URL: $SERVER_URL"
  
  # Use provided token or get a new one
  local token=$TOKEN
  local auth_token=$TOKEN
  local space=$SPACE
  local registry=$REGISTRY
  local epoch=$(date +%s)
  
  # If no token provided, login and get one
  if [ -z "$token" ]; then
    # Login and get token using Gitness admin credentials (.local.env has admin@gitness.io/changeit)
    log "Logging in to Gitness..."
    token=$(curl -s -X 'POST' "http://$SERVER_URL/api/v1/login?include_cookie=false" \
      -H 'accept: application/json' \
      -H 'Content-Type: application/json' \
      -d '{
        "login_identifier": "admin@gitness.io",
        "password": "changeit"
      }' | jq -r '.access_token')

    # Verify we got a valid token
    if [ "$token" == "null" ] || [ -z "$token" ]; then
      log "ERROR: Failed to authenticate with Gitness. Check credentials."
      exit 1
    fi
    log "Authentication successful"

    # Get PAT
    log "Getting Personal Access Token..."
    local pat=$(curl -s -X 'POST' "http://$SERVER_URL/api/v1/user/tokens" \
      -H 'accept: application/json' \
      -H 'Content-Type: application/json' \
      -H "Authorization: Bearer $token" \
      -d "{\"uid\":\"code_token_$epoch\"}" | jq -r '.access_token')
      
    # Select the right authentication token
    if [ "$pat" == "null" ] || [ -z "$pat" ]; then
      log "WARNING: PAT token is null or empty, using login token instead"
      auth_token=$token
    else
      auth_token=$pat
    fi
  fi
  
  # Create space if not provided
  if [ -z "$space" ]; then
    space="npm_space_$epoch"
    log "Creating space... $space"
    curl -s -X 'POST' "http://$SERVER_URL/api/v1/spaces" \
      -H 'Content-Type: application/json' \
      -H "Authorization: Bearer $token" \
      -H 'Accept: application/json' \
      -d "{\"description\": \"npm conformance test\", \"identifier\": \"$space\",\"is_public\": true, \"parent_ref\": \"\"}"
  fi

  # Create registry if not provided
  if [ -z "$registry" ]; then
    registry="npm_registry_$epoch"
    log "Creating registry: $registry"
    curl -s -X 'POST' "http://$SERVER_URL/api/v1/registry" \
      -H 'Content-Type: application/json' \
      -H "Authorization: Bearer $token" \
      -H 'Accept: application/json' \
      -d "{\"config\":{\"type\": \"VIRTUAL\"}, \"description\": \"\", \"identifier\": \"$registry\", \"packageType\": \"NPM\",\"parentRef\": \"$space\"}"
  else
    log "Using existing registry: $registry"
  fi

  # Handle namespace format - check if it contains a slash which indicates space/registry format
  local namespace_value="$space"
  if [[ "$space" == *"/"* ]]; then
    # If space contains a slash, it's already in space/registry format
    # Extract the space part before the slash
    local space_part=$(echo "$space" | cut -d'/' -f1)
    # Extract the registry part after the slash
    local registry_part=$(echo "$space" | cut -d'/' -f2)
    
    # Use the extracted parts
    space="$space_part"
    if [ -n "$registry_part" ]; then
      registry="$registry_part"
    fi
    
    # Set namespace to just the space name to avoid duplicate registry in path
    namespace_value="$space"
    log "Using namespace: $namespace_value (space: $space, registry: $registry)"
  else
    # If no slash, use just the space name
    namespace_value="$space"
    log "Using namespace: $namespace_value (space: $space, registry: $registry)"
  fi

  # Create environment file
  ENV_FILE="/tmp/npm_env.sh"
  echo "# npm test environment variables" > "$ENV_FILE"
  echo "export REGISTRY_ROOT_URL=\"http://$SERVER_URL\"" >> "$ENV_FILE"
  echo "export REGISTRY_USERNAME=\"admin@gitness.io\"" >> "$ENV_FILE"
  echo "export REGISTRY_PASSWORD=\"$auth_token\"" >> "$ENV_FILE"
  echo "export REGISTRY_NAMESPACE=\"$namespace_value\"" >> "$ENV_FILE"
  echo "export REGISTRY_NAME=\"$registry\"" >> "$ENV_FILE"
  echo "export DEBUG=\"$DEBUG\"" >> "$ENV_FILE"
  chmod +x "$ENV_FILE"
  
  # Export variables for immediate use
  export REGISTRY_ROOT_URL="http://$SERVER_URL"
  export REGISTRY_USERNAME="admin@gitness.io"
  export REGISTRY_PASSWORD="$auth_token"
  export REGISTRY_NAMESPACE="$namespace_value"
  export REGISTRY_NAME="$registry"
  export DEBUG="$DEBUG"

  log "Setup complete. Environment variables written to $ENV_FILE"
}



# Main execution flow
setup_environment

log "To run tests, use: go test -v ./registry/tests/npm"

# Exit with success status
exit 0
