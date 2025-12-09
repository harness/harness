#!/bin/bash
set -e

function createSpace {
  echo "Creating space... $2"
  curl --location --request POST "http://$1/api/v1/spaces" \
  --header 'Content-Type: application/json' \
  --header 'Authorization: Bearer '"$3" \
  --header 'Accept: application/json' \
  --data "{\"description\": \"corformance test\", \"identifier\": \"$2\",\"is_public\": true, \"parent_ref\": \"\"}"
}


function createRegistry {
   echo "Creating registry: $2"
   curl --location "http://$1/api/v1/registry?space_ref=$3" \
   --header 'Content-Type: application/json' \
   --header 'Authorization: Bearer '"$4" \
   --header 'Accept: application/json' \
   --data "{\"config\":{\"type\": \"VIRTUAL\"}, \"description\": \"mydesc\", \"identifier\": \"$2\", \"packageType\": \"DOCKER\",\"parentRef\": \"$3\"}"
}

function login {
    # Define the URL and request payload
    url="http://$1/api/v1/login?include_cookie=false"
    payload='{
      "login_identifier": "admin",
      "password": "changeit"
    }'

    # Make the curl call and capture the response
    response=$(curl -s -X 'POST' "$url" -H 'accept: application/json' -H 'Content-Type: application/json' -d "$payload")

    # Extract the access_token using jq
    access_token=$(echo "$response" | jq -r '.access_token')

    # Check if jq command succeeded
    if [ $? -ne 0 ]; then
      echo "Failed to parse access_token"
      exit 1
    fi

    # Print the access_token
#    echo "Access Token: $access_token"
    echo "$access_token"
}

function getPat {
    # Define the URL and request payload
    url="http://$1/api/v1/user/tokens"
    payload="{\"uid\":\"code_token_$2\"}"

    # Make the curl call and capture the response
    response=$(curl -s -X 'POST' "$url" -H 'accept: application/json' -H 'Content-Type: application/json' -H 'Cookie: token='"$3" -d "$payload")

    # Extract the access_token using jq
    access_token=$(echo "$response" | jq -r '.access_token')

    # Check if jq command succeeded
    if [ $? -ne 0 ]; then
      echo "Failed to parse access_token"
      exit 1
    fi

    # Print the access_token
#    echo "Access Token: $access_token"
    echo "$access_token"
}


export epoch=$(date +%s)

export space="Space_$epoch"
export space_lower=$(echo $space | tr '[:upper:]' '[:lower:]')
export conformance="conformance_$epoch"
export crossmount="crossmount_$epoch"

export token=$(login $1)
export pat=$(getPat $1 $epoch $token)
createSpace $1 $space $token
createRegistry $1 $conformance $space $token
createRegistry $1 $crossmount $space $token

# Save current directory
export CURRENT_DIR=$(pwd)

bash "./registry/tests/scripts/oci_tests.sh" $1
bash "./registry/tests/scripts/maven_tests.sh" $1
bash "./registry/tests/scripts/cargo_tests.sh" $1
bash "./registry/tests/scripts/go_tests.sh" $1
bash "./registry/tests/scripts/npm_tests.sh" $1


cd "$CURRENT_DIR"

echo "All tests passed successfully"
TEST_EXIT_CODE=0

# Cleanup temporary directory
rm -rf "$TEMP_DIR"

# Return the test exit code
exit $TEST_EXIT_CODE
