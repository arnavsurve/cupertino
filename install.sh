#!/bin/bash
set -euo pipefail

# Cupertino installer - macOS only package manager
# Usage: curl -fsSL https://raw.githubusercontent.com/user/cupertino/main/install.sh | bash

abort() {
  printf "%s\n" "$@" >&2
  exit 1
}

# Check if script is run with force-interactive mode in CI
if [[ -n "${CI-}" && -n "${INTERACTIVE-}" ]]; then
  abort "Cannot run force-interactive mode in CI."
fi

# Check if both INTERACTIVE and NONINTERACTIVE are set
if [[ -n "${INTERACTIVE-}" && -n "${NONINTERACTIVE-}" ]]; then
  abort 'Both $INTERACTIVE and $NONINTERACTIVE are set. Please unset at least one variable and try again.'
fi

# Determine if running interactively
if [[ -z "${NONINTERACTIVE-}" ]]; then
  if [[ -n "${CI-}" ]]; then
    echo "Running in non-interactive mode because CI is set."
    NONINTERACTIVE=1
  elif [[ ! -t 0 ]]; then
    if [[ -z "${INTERACTIVE-}" ]]; then
      echo "Running in non-interactive mode because stdin is not a TTY."
      NONINTERACTIVE=1
    fi
  fi
fi

# Check for macOS
if [[ "$(uname)" != "Darwin" ]]; then
  abort "Cupertino only supports macOS."
fi

# Don't run as root
if [[ "${EUID:-${UID}}" == "0" ]]; then
  abort "Don't run this script as root!"
fi

# Set up colors for output
if [[ -t 1 ]]; then
  tty_blue="$(printf '\033[34m')"
  tty_bold="$(printf '\033[1m')"
  tty_reset="$(printf '\033[0m')"
else
  tty_blue=""
  tty_bold=""
  tty_reset=""
fi

ohai() {
  printf "${tty_blue}==>${tty_bold} %s${tty_reset}\n" "$*"
}

# Get user info
USER="${USER:-$(id -un)}"

# Set prefix (allow override via environment)
CUPERTINO_PREFIX="${CUPERTINO_PREFIX:-/opt/cupertino}"

# Check for sudo access
have_sudo_access() {
  if [[ ! -x "/usr/bin/sudo" ]]; then
    return 1
  fi

  if [[ -n "${NONINTERACTIVE-}" ]]; then
    /usr/bin/sudo -n -l mkdir &>/dev/null
  else
    /usr/bin/sudo -v && /usr/bin/sudo -l mkdir &>/dev/null
  fi
}

execute_sudo() {
  ohai "/usr/bin/sudo $*"
  if ! /usr/bin/sudo "$@"; then
    abort "Failed to execute: sudo $*"
  fi
}

# Check if we need to create directories
need_setup=false
if [[ ! -d "${CUPERTINO_PREFIX}" ]]; then
  need_setup=true
elif [[ ! -w "${CUPERTINO_PREFIX}" ]]; then
  need_setup=true
fi

if [[ "${need_setup}" == "true" ]]; then
  ohai "Cupertino will be installed to:"
  echo "${CUPERTINO_PREFIX}/bin/cupertino"
  echo "${CUPERTINO_PREFIX}/packages/"
  echo "${CUPERTINO_PREFIX}/cache/"
  echo ""

  if [[ -z "${NONINTERACTIVE-}" ]]; then
    echo "Press RETURN to continue or any other key to abort:"
    read -r -n 1 -s key
    if [[ "${key}" != "" ]]; then
      exit 1
    fi
  fi

  # Check for sudo access
  if ! have_sudo_access; then
    abort "Need sudo access to create ${CUPERTINO_PREFIX}"
  fi

  # Create directory structure
  ohai "Creating Cupertino directories..."
  execute_sudo mkdir -p "${CUPERTINO_PREFIX}"/{bin,packages,cache}

  # Set proper ownership (user:admin on macOS)
  ohai "Setting up permissions..."
  execute_sudo chown -R "${USER}:admin" "${CUPERTINO_PREFIX}"
  execute_sudo chmod -R 755 "${CUPERTINO_PREFIX}"
fi

# TODO: Download and install cupertino binary
# For now, we'll just create a placeholder
ohai "Installing Cupertino binary..."
if [[ ! -f "${CUPERTINO_PREFIX}/bin/cupertino" ]]; then
  # This would download the actual binary in production
  # curl -L "https://github.com/user/cupertino/releases/latest/download/cupertino-darwin" \
  #   -o "${CUPERTINO_PREFIX}/bin/cupertino"
  
  # For now, create a placeholder
  echo "#!/bin/bash" > "${CUPERTINO_PREFIX}/bin/cupertino"
  echo 'echo "Cupertino placeholder - replace with actual binary"' >> "${CUPERTINO_PREFIX}/bin/cupertino"
  chmod +x "${CUPERTINO_PREFIX}/bin/cupertino"
fi

# Set up PATH
ohai "Setting up PATH..."

# Try to add to /etc/paths.d for system-wide PATH (like Homebrew does)
if [[ -d "/etc/paths.d" && -x "$(command -v tee)" ]]; then
  if ! grep -q "${CUPERTINO_PREFIX}/bin" /etc/paths.d/cupertino 2>/dev/null; then
    echo "${CUPERTINO_PREFIX}/bin" | execute_sudo tee /etc/paths.d/cupertino >/dev/null
    execute_sudo chown root:wheel /etc/paths.d/cupertino
    execute_sudo chmod a+r /etc/paths.d/cupertino
    ohai "Added to /etc/paths.d/cupertino for automatic PATH setup"
  fi
else
  # Fallback to manual shell setup
  case "${SHELL##*/}" in
    zsh)  shell_rcfile="${HOME}/.zshrc" ;;
    bash) shell_rcfile="${HOME}/.bash_profile" ;;
    *)    shell_rcfile="${HOME}/.profile" ;;
  esac

  if [[ -f "${shell_rcfile}" ]] && ! grep -q "cupertino" "${shell_rcfile}"; then
    echo "" >> "${shell_rcfile}"
    echo "# Cupertino package manager" >> "${shell_rcfile}"
    echo "export PATH=\"${CUPERTINO_PREFIX}/bin:\$PATH\"" >> "${shell_rcfile}"
    ohai "Added Cupertino to PATH in ${shell_rcfile}"
  fi
fi

# Success message
ohai "Installation successful!"
echo ""
echo "Cupertino has been installed to ${CUPERTINO_PREFIX}"
echo ""

# Check if cupertino is in PATH
if [[ ":${PATH}:" != *":${CUPERTINO_PREFIX}/bin:"* ]]; then
  ohai "Next steps:"
  echo "- Add Cupertino to your PATH by running:"
  echo "    export PATH=\"${CUPERTINO_PREFIX}/bin:\$PATH\""
  echo "- Or restart your terminal to pick up the PATH changes"
  echo ""
fi

ohai "Run 'cupertino help' to get started!"
