#!/bin/sh
osInfoScript={{ .OSInfoScript }}

eval "$osInfoScript"

# Function to install Node.js and npm
install_nodejs() {
  echo "Checking if Node.js is installed..."
  if ! command -v node >/dev/null 2>&1; then
    echo "Installing Node.js for $(distro)"
    case "$(distro)" in
      debian)
        export DEBIAN_FRONTEND=noninteractive
        # Install Node.js 18.x LTS
        curl -fsSL https://deb.nodesource.com/setup_18.x | bash -
        apt-get install -y nodejs
        ;;
      fedora)
        dnf install -y nodejs npm
        ;;
      opensuse)
        zypper install -y nodejs18 npm18
        ;;
      alpine)
        apk add --no-cache nodejs npm
        ;;
      arch)
        pacman -Syu --noconfirm nodejs npm
        ;;
      freebsd)
        pkg install -y node18 npm
        ;;
      *)
        echo "Unsupported distribution: $(distro). Please install Node.js manually."
        exit 1
        ;;
    esac
    echo "Node.js installation completed."
  else
    echo "Node.js is already installed."
  fi
  # Verify installation
  if command -v node >/dev/null && command -v npm >/dev/null; then
      echo "Successfully installed Node.js and npm"
      return 0
  else
      echo "Failed to install Node.js and npm"
      return 1
  fi
}

# Function to install Claude Code CLI
install_claude_code() {
    echo "Installing Claude Code CLI..."
    npm install -g @anthropic-ai/claude-code
    if command -v claude >/dev/null; then
        echo "Claude Code CLI installed successfully!"
        claude --version
        return 0
    else
        echo "ERROR: Claude Code CLI installation failed!"
        return 1
    fi
}

# Install required dependencies
if ! install_nodejs; then
    echo "ERROR: Failed to install Node.js dependencies"
    exit 1
fi

# Install Claude agent
if ! install_claude_code; then
    echo "ERROR: Failed to install Claude Code CLI"
    exit 1
fi
