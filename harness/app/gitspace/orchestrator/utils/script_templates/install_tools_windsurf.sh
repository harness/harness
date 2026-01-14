#!/bin/sh

osInfoScript={{ .OSInfoScript }}

eval "$osInfoScript"

echo "Checking if tar is installed..."
if ! command -v tar >/dev/null 2>&1; then
  echo "Installing tar for $(distro)"
  case "$(distro)" in
    opensuse)
      zypper install -y tar
      zypper install -y gzip
      ;;
    *)
      echo "Distribution: $distro. tar installation not required"
      exit 1
      ;;
  esac
  echo "tar installation completed."
fi