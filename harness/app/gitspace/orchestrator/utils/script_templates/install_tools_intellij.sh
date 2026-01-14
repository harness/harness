#!/bin/sh

osInfoScript={{ .OSInfoScript }}

# provide distro func to evaluate os distribution
eval "$osInfoScript"

# finding os distribution
os_dist=$(distro)

# List of command-line tools to check and install based on the distribution
debian_tools="procps tar unzip curl libxext6 libxrender1 libxtst6 libxi6 freetype* procps"
fedora_tools="procps-ng tar unzip curl libxext6 libxrender1 libxtst6 libxi6 freetype* procps"
alpine_tools="tar unzip curl libxext libxrender libxtst libxi freetype* procps gcompat"

# Install tool based on the distribution
install_tool() {
  tool=$1

  case "$os_dist" in
    debian|ubuntu)
    export DEBIAN_FRONTEND=noninteractive && apt-get update && apt-get install -y "$tool"
      ;;
    fedora)
      dnf install -y "$tool"
      ;;
    opensuse)
      zypper install -y "$tool"
      ;;
    alpine)
      apk add --no-cache "$tool"
      ;;
    arch)
      pacman -Syu --noconfirm "$tool"
      ;;
    freebsd)
      pkg install -y "$tool"
      ;;
    *)
      echo "Unsupported OS distribution: $os_dist"
      exit 1
      ;;
  esac
}

install_tools() {
  case "$os_dist" in
      debian|ubuntu)
        tools="$debian_tools"
        ;;
      fedora)
        tools="$fedora_tools"
        ;;
      alpine)
        tools="$alpine_tools"
        ;;
      *)
        echo "no tools to install,  distribution: $os_dist"
        ;;
    esac

  for tool in $tools; do
      if ! command -v "$tool" >/dev/null 2>&1; then
        echo "$tool is not installed. Installing..."
        install_tool "$tool"
      else
        echo "$tool is already installed."
      fi
  done
}

install_tools