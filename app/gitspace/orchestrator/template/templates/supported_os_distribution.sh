#!/bin/sh

osInfoScript={{ .OSInfoScript }}

eval "$osInfoScript"

case "$(distro)" in
  debian|fedora|opensuse)
    echo "Detected $(distro) distribution"
    ;;
  *)
    echo "Unsupported distribution: $distro." >&2
    exit 1
    ;;
esac
