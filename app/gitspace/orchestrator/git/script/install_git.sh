#!/bin/sh

# Detect OS type
os() {
  uname="$(uname)"
  case $uname in
    Linux) echo linux ;;
    Darwin) echo macos ;;
    FreeBSD) echo freebsd ;;
    *) echo "$uname" ;;
  esac
}

# Detect Linux distro type
distro() {
  local os_name
  os_name=$(os)

  if [ "$os_name" = "macos" ] || [ "$os_name" = "freebsd" ]; then
    echo "$os_name"
    return
  fi

  if [ -f /etc/os-release ]; then
    (
      . /etc/os-release
      if [ "${ID_LIKE-}" ]; then
        for id_like in $ID_LIKE; do
          case "$id_like" in debian | fedora | opensuse | arch)
            echo "$id_like"
            return
            ;;
          esac
        done
      fi

      echo "$ID"
    )
    return
  fi
}

# Print a human-readable name for the OS/distro
distro_name() {
  if [ "$(uname)" = "Darwin" ]; then
    echo "macOS v$(sw_vers -productVersion)"
    return
  fi

  if [ -f /etc/os-release ]; then
    (
      . /etc/os-release
      echo "$PRETTY_NAME"
    )
    return
  fi

  uname -sr
}


# Install Git if not already installed
install_git() {
  # Check if Git is installed
  if ! command -v git >/dev/null 2>&1; then
    echo "Git is not installed. Installing Git..."

    case "$(distro)" in
      debian | ubuntu)
        apt-get update && apt-get install -y git
        ;;
      fedora | centos | rhel)
        dnf install -y git
        ;;
      opensuse)
        zypper install -y git
        ;;
      alpine)
        apk add git
        ;;
      arch | manjaro)
        pacman -Sy --noconfirm git
        ;;
      freebsd)
        pkg install -y git
        ;;
      macos)
        brew install git
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
