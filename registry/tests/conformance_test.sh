#!/bin/bash
set -e

echo "get the conformance testing code..."
git clone https://github.com/opencontainers/distribution-spec.git

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
   curl --location "http://$1/api/v1/registry" \
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


epoch=$(date +%s)

space="Space_$epoch"
space_lower=$(echo $space | tr '[:upper:]' '[:lower:]')
conformance="conformance_$epoch"
crossmount="crossmount_$epoch"

token=$(login $1)
pat=$(getPat $1 $epoch $token)
createSpace $1 $space $token
createRegistry $1 $conformance $space $token
createRegistry $1 $crossmount $space $token

echo "run conformance test..."
export OCI_ROOT_URL="http://$1"
export OCI_NAMESPACE="$space_lower/$conformance/testrepo"
export OCI_DEBUG="true"

export OCI_TEST_PUSH=1
export OCI_TEST_PULL=1
export OCI_TEST_CONTENT_DISCOVERY=1
export OCI_TEST_CONTENT_MANAGEMENT=1
export OCI_CROSSMOUNT_NAMESPACE="$space_lower/$crossmount/testrepo"
export OCI_AUTOMATIC_CROSSMOUNT="false"

export OCI_USERNAME="admin"
export OCI_PASSWORD="$pat"
cd ./distribution-spec/conformance
go test .