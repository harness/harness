#!/bin/sh

osInfoScript={{ .OSInfoScript }}

eval "$osInfoScript"

# Check if curl is installed
echo "Checking if curl is installed..."

if ! command -v curl >/dev/null 2>&1; then
  echo "Installing curl for $(distro)"
  case "$(distro)" in
    debian)
      apt update && apt install -y curl
      ;;
    fedora)
      dnf install -y curl
      ;;
    opensuse)
      zypper install -y curl
      ;;
    alpine)
      apk add --no-cache curl
      ;;
    arch)
      pacman -Syu --noconfirm curl
      ;;
    freebsd)
      pkg install -y curl
      ;;
    *)
      echo "Unsupported distribution: $(distro)."
      exit 1
      ;;
  esac
  echo "Curl installation completed."
fi

if ! command -v npm >/dev/null 2>&1; then
  echo "Installing npm..."
  case "$(distro)" in
    alpine)
      echo "Detected Alpine Linux. Installing npm..."
      apk update
      apk add nodejs npm
      ;;
    freebsd)
      echo "Detected FreeBSD. Installing npm..."
      pkg update
      pkg install -y node
      ;;
    *)
      echo "Distribution: $(distro). npm installation not required"
      ;;
  esac
  echo "npm installation completed."
fi