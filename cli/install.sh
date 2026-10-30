#!/usr/bin/env bash
#
# Installer for the epos-cli CLI.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/EPOS-ERIC/epos-platform/main/cli/install.sh | bash
#
# The script will:
# 1. Detect the user's OS and architecture.
# 2. Check if the CLI is already installed and find its version.
# 3. Fetch the latest version from GitHub releases.
# 4. Compare versions and check for breaking changes (a major version increase).
# 5. If a breaking change is detected, it will ask for user confirmation before proceeding.
# 6. Download the appropriate binary and place it in a common PATH directory.
# 7. Remove old installation if updating from a different directory.
# ------------------------------------------------------------------------------

# --- Configuration ---
# Exit on any error, undefined variable, or pipe failure.
set -euo pipefail

# GitHub repository for the CLI tool.
readonly REPO="EPOS-ERIC/epos-platform"
# The name of the binary to be installed.
readonly BINARY_NAME="epos-cli"
# GitHub API URL for the latest release.
readonly GITHUB_API_URL="https://api.github.com/repos/${REPO}/releases/latest"


# --- Utilities and Logging ---
# Pretty logging functions with color and bold text.
BOLD=$(tput bold 2>/dev/null || echo "")
BLUE=$(tput setaf 4 2>/dev/null || echo -e "\e[34m")
CYAN=$(tput setaf 6 2>/dev/null || echo -e "\e[36m")
GREEN=$(tput setaf 2 2>/dev/null || echo -e "\e[32m")
YELLOW=$(tput setaf 3 2>/dev/null || echo -e "\e[33m")
RED=$(tput setaf 1 2>/dev/null || echo -e "\e[31m")
NC=$(tput sgr0 2>/dev/null || echo -e "\e[0m") # No Color

log_step() {
    printf '%s[STEP]%s  %s\n' "${CYAN}" "${NC}" "$*"
}

log_info() {
    printf '%s[INFO]%s  %s\n' "${BLUE}" "${NC}" "$*"
}

log_warn() {
    printf '%s%s[WARN]%s  %s\n' "${YELLOW}" "${BOLD}" "${NC}" "$*"
}

log_error() {
    printf '%s%s[ERROR] %s%s\n' "${RED}" "${BOLD}" "$*" "${NC}" >&2
    exit 1
}

log_done() {
    printf '%s[DONE]%s  %s\n' "${GREEN}" "${NC}" "$*"
}


# --- Core Functions ---

# get_platform_and_arch detects the operating system and CPU architecture.
# It sets the global variables OS and ARCH.
get_platform_and_arch() {
    local uname_s
    uname_s="$(uname -s | tr '[:upper:]' '[:lower:]')"

    case "${uname_s}" in
        darwin)
            OS="darwin"
            ;;
        linux)
            OS="linux"
            ;;
        *)
            log_error "Unsupported operating system: ${uname_s}. Only macOS and Linux are supported."
            ;;
    esac

    local uname_m
    uname_m="$(uname -m)"
    case "${uname_m}" in
        x86_64 | amd64)
            ARCH="amd64"
            ;;
        arm64 | aarch64)
            ARCH="arm64"
            ;;
        *)
            log_error "Unsupported architecture: ${uname_m}. Only x86_64/amd64 and arm64/aarch64 are supported."
            ;;
    esac

    if [ "${OS}" = "linux" ] && [ "${ARCH}" = "arm64" ]; then
        log_error "Unsupported platform: linux-arm64. No release binary is available."
    fi
}

# get_current_install_path returns the full path to the currently installed binary.
# Returns an empty string if not found.
get_current_install_path() {
    if command -v "${BINARY_NAME}" &>/dev/null; then
        command -v "${BINARY_NAME}"
    else
        echo ""
    fi
}

# get_local_version checks for an existing installation of the binary and extracts its version.
# Returns a semantic version string (e.g., "1.2.3"), a DEV_BUILD prefixed string for
# non-standard versions, or an empty string if not found.
get_local_version() {
    if ! command -v "${BINARY_NAME}" &>/dev/null; then
        echo ""
        return
    fi

    local version_output
    # Try the standard '--version' flag first, then the 'version' subcommand as a fallback.
    if ! version_output="$(${BINARY_NAME} --version 2>/dev/null || ${BINARY_NAME} version 2>/dev/null)"; then
        log_warn "Found '${BINARY_NAME}' but could not determine its version."
        echo ""
        return
    fi

    # Try to extract a clean semver (e.g., 0.1.5) from a more complex string (e.g., v0.1.5-commit-hash)
    local semver
    semver=$(echo "${version_output}" | grep -o -E '[0-9]+\.[0-9]+\.[0-9]+' | head -n 1)

    if [ -n "${semver}" ]; then
        echo "${semver}"
    elif [ -n "${version_output}" ]; then
        # If no standard semver is found but there is version output, treat it as a dev build.
        # Prefix with DEV_BUILD: to make it easy to check in the main logic.
        echo "DEV_BUILD:${version_output}"
    else
        echo ""
    fi
}

# get_latest_release_info fetches release data from the GitHub API.
# It sets the global variables LATEST_VERSION and DOWNLOAD_URL.
get_latest_release_info() {
    local api_json
    if ! api_json="$(curl -fsSL "${GITHUB_API_URL}")"; then
        log_error "Failed to fetch release information from GitHub."
    fi

    # Extract the tag name (version) from the API response.
    local tag_name
    tag_name=$(echo "${api_json}" | grep -o '"tag_name": *"[^"]*"' | cut -d'"' -f4)
    if [ -z "${tag_name}" ]; then
        log_error "Could not find the latest release tag in the GitHub API response."
    fi
    LATEST_VERSION=$(echo "${tag_name}" | sed 's/^v//')

    # Construct the expected asset name and find its download URL.
    local asset_name="${BINARY_NAME}-${OS}-${ARCH}"
    DOWNLOAD_URL=$(echo "${api_json}" | grep -o '"browser_download_url": *"[^"]*'"${asset_name}"'"' | cut -d'"' -f4)

    if [ -z "${DOWNLOAD_URL}" ]; then
        log_error "Could not find the download URL for asset '${asset_name}' in the latest release."
    fi
}

# get_install_dir determines the best directory to install the binary.
# It prioritizes system-wide directories over user-local directories.
get_install_dir() {
    # Preferred installation directory.
    # On both macOS and Linux, /usr/local/bin is a common choice.
    if [ -d "/usr/local/bin" ]; then
        echo "/usr/local/bin"
        return
    fi

    # Fallback for user-local installation.
    echo "${HOME}/.local/bin"
}

# remove_old_installation removes the old binary if it exists in a different location
# than the new installation directory.
remove_old_installation() {
    local old_path=$1
    local new_install_dir=$2
    local new_install_path="${new_install_dir}/${BINARY_NAME}"

    # Don't remove if it's the same location
    if [ "${old_path}" = "${new_install_path}" ]; then
        return
    fi

    log_info "Removing old installation from: ${old_path}"
    
    local old_dir
    old_dir=$(dirname "${old_path}")
    
    if [ ! -w "${old_dir}" ]; then
        if command -v sudo &>/dev/null; then
            sudo rm -f "${old_path}"
        else
            log_warn "Could not remove old installation at ${old_path} (no sudo available)"
            log_warn "You may want to manually remove it later"
            return
        fi
    else
        rm -f "${old_path}"
    fi
    
    log_info "Old installation removed successfully"
}

# provide_path_instructions detects the user's shell and gives specific
# instructions on how to add the installation directory to the PATH.
provide_path_instructions() {
    local install_dir=$1
    local shell_name
    shell_name=$(basename "${SHELL}")
    local profile_file

    if [ "${shell_name}" = "zsh" ]; then
        profile_file="${HOME}/.zshrc"
    elif [ "${shell_name}" = "bash" ]; then
        # macOS login shells use .bash_profile, Linux often uses .bashrc
        if [[ "${OS}" == "darwin" ]] && [ -f "${HOME}/.bash_profile" ]; then
            profile_file="${HOME}/.bash_profile"
        else
            profile_file="${HOME}/.bashrc"
        fi
    else
        # Fallback for less common shells like fish, dash, etc.
        log_warn "The directory '${install_dir}' is not in your PATH."
        log_warn "Please add it to your shell's configuration file to run '${BINARY_NAME}' directly."
        return
    fi

    local export_cmd="export PATH=\"${install_dir}:\$PATH\""

    log_warn "The directory '${install_dir}' is not in your PATH."
    log_info "Add it to your ${shell_name} configuration:"
    log_info "  echo '${export_cmd}' >> ${profile_file}"
    log_info "Then restart your terminal or run: source ${profile_file}"
}

# --- Main Execution Logic ---
main() {
    log_info "${BINARY_NAME} installer"

    # 1. Determine OS, Architecture, and versions
    log_step "Step 1: Checking your system and existing installation..."
    get_platform_and_arch
    log_info "Detected Platform: ${OS}-${ARCH}"

    local current_install_path
    current_install_path=$(get_current_install_path)
    
    local local_version
    local_version=$(get_local_version)
    
    if [ -n "${current_install_path}" ]; then
        log_info "Found existing installation at: ${current_install_path}"
    fi
    
    log_info "Fetching latest release from GitHub..."
    get_latest_release_info # This sets LATEST_VERSION and DOWNLOAD_URL
    

    # 2. Decision Logic: Install, Update, or Exit
    log_step "Step 2: Comparing versions..."
    if [ -z "${local_version}" ]; then
        log_info "${BINARY_NAME} is not installed."
        log_info "Latest version available: ${LATEST_VERSION}"
    elif [[ "${local_version}" == DEV_BUILD:* ]]; then
        local raw_version="${local_version#DEV_BUILD:}" # Remove prefix
        log_info "Found a local development build: '${raw_version}'"
        log_info "Latest available release:      ${LATEST_VERSION}"
        log_warn "Cannot automatically compare a development build with the latest release."

        printf "  ${BOLD}Do you want to replace your local build with the official ${LATEST_VERSION} release? [y/N]: ${NC}"
        read -r answer < /dev/tty
        if [[ ! "$answer" =~ ^[Yy]$ ]]; then
            log_info "Installation cancelled by user."
            exit 0
        fi
    else
        log_info "Current installed version: ${local_version}"
        log_info "Latest available version:  ${LATEST_VERSION}"

        if [ "${local_version}" = "${LATEST_VERSION}" ]; then
            log_done "You already have the latest version installed. Nothing to do."
            exit 0
        fi
        
        # Check if the local version is newer than the latest release to prevent downgrades.
        # We use `sort -V` which correctly handles semantic versioning. The oldest version comes first.
        if [ "$(printf '%s\n' "${LATEST_VERSION}" "${local_version}" | sort -V | head -n 1)" = "${LATEST_VERSION}" ]; then
            log_info "Your local version (${local_version}) is newer than the latest release (${LATEST_VERSION})."
            log_done "No update necessary. Keeping your current version."
            exit 0
        fi

        # Breaking change check: compare major versions.
        local local_major_version
        local_major_version=$(echo "${local_version}" | cut -d'.' -f1)
        local latest_major_version
        latest_major_version=$(echo "${LATEST_VERSION}" | cut -d'.' -f1)

        if [ "${latest_major_version}" -gt "${local_major_version}" ]; then
            log_warn "Major version upgrade: ${local_version} -> ${LATEST_VERSION}."
            log_warn "Updating across a major version may break existing environments."
            
            printf "  ${BOLD}Do you want to proceed with the update? [y/N]: ${NC}"
            read -r answer < /dev/tty
            if [[ ! "$answer" =~ ^[Yy]$ ]]; then
                log_info "Installation cancelled by user."
                exit 0
            fi
        fi
    fi

    # 3. Determine install directory and check PATH
    log_step "Step 3: Preparing for installation..."
    local install_dir
    install_dir=$(get_install_dir)
    mkdir -p "${install_dir}"
    log_info "Binary will be installed to: ${install_dir}"

    local path_needs_update=false
    case ":${PATH}:" in
        *":${install_dir}:"*)
            ;; # It's in the path, do nothing
        *)
            path_needs_update=true
            provide_path_instructions "${install_dir}"
            ;;
    esac

    # 4. Download and install
    log_step "Step 4: Downloading and Installing..."
    local temp_file
    temp_file=$(mktemp)
    local install_path="${install_dir}/${BINARY_NAME}"

    log_info "Downloading from ${DOWNLOAD_URL}"
    if ! curl --fail --progress-bar -L "${DOWNLOAD_URL}" -o "${temp_file}"; then
        rm "${temp_file}"
        log_error "Download failed. Please check your network connection or the URL."
    fi

    chmod +x "${temp_file}"
    log_info "Moving binary to ${install_path}"
    if [ ! -w "${install_dir}" ]; then
        log_info "Administrator privileges are required to write to ${install_dir}."
        if command -v sudo &>/dev/null; then
            sudo mv "${temp_file}" "${install_path}"
        else
            log_error "sudo not found. Cannot move file to ${install_path}."
        fi
    else
        mv "${temp_file}" "${install_path}"
    fi

    # 5. Remove old installation only after the replacement is installed.
    if [ -n "${current_install_path}" ] && [ "${current_install_path}" != "${install_path}" ]; then
        log_step "Step 5: Cleaning up old installation..."
        remove_old_installation "${current_install_path}" "${install_dir}"
    fi
    
    log_done "${BINARY_NAME} version ${LATEST_VERSION} installed to ${install_path}."
    if [ "${path_needs_update}" = true ]; then
        log_warn "Restart your terminal for PATH changes to take effect."
    fi
    log_info "Run '${BINARY_NAME} --help' to get started."
}

# Run the main function.
main
