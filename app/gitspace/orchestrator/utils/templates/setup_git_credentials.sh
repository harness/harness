#!/bin/sh
url={{ .CloneURLWithCreds }}

#run git operation to cache the credential in memory
if [ -z "$url" ]; then
    echo "setting up without credentials"
else
    git config --global credential.helper 'cache --timeout=2592000'
    touch .gitcontext
    echo "url="$url >> .gitcontext
    echo "" >> .gitcontext

    cat .gitcontext | git credential approve
    rm .gitcontext
fi
