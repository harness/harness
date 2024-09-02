#!/bin/sh
name={{ .Name }}
password={{ .Password }}
email={{ .Email }}
host={{ .Host }}
protocol={{ .Protocol }}
path={{ .Path }}

#run git operation to cache the credential in memory
if [ -z "$password" ]; then
    echo "setting up without credentials"
else
    git config --global credential.helper 'cache --timeout=2592000'
    git config --global user.email "$email"
    git config --global user.name "$name"
    touch .gitcontext
    echo "host="$host >> .gitcontext
    echo "protocol="$protocol >> .gitcontext
    echo "path="$path >> .gitcontext
    echo "username="$email >> .gitcontext
    echo "password="$password >> .gitcontext
    echo "" >> .gitcontext

    cat .gitcontext | git credential approve
    rm .gitcontext
fi
