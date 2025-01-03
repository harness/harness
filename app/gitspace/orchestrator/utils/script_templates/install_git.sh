#!/bin/sh

osInfoScript={{ .OSInfoScript }}

eval "$osInfoScript"

# Install Git if not already installed
install_git() {
  # Check if Git is installed
  if ! command -v git >/dev/null 2>&1; then
    echo "Git is not installed. Installing Git..."
    case "$(distro)" in
        debian)
            export DEBIAN_FRONTEND=noninteractive && apt-get update && apt-get install -y git
          ;;
        fedora)
            dnf install -y git
          ;;
        opensuse)
            zypper install -y git
          ;;
        alpine)
          apk add git
          ;;
        arch)
          pacman -Sy --noconfirm git
          ;;
        freebsd)
          pkg install -y git
          ;;
        *)
          echo "Unsupported OS for automatic Git installation."
          return 1
          ;;
      esac
  fi

  # Verify installation
  if ! command -v git >/dev/null 2>&1; then
    echo "Git is not installed. Exiting..."
    exit 1
  else
    echo "Git is installed."
  fi
}

# Run the installation function
install_git
