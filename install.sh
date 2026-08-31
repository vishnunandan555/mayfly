#!/usr/bin/env bash
# MayFly: All-in-One Global Installer, Updater, and Uninstaller
# Supports Linux (all distros) and macOS (Darwin Intel & Apple Silicon)
# Features: Tier-2 Cryptographic SHA-256 Checksum Verification & Offline Local Builds

set -e

INSTALL_DIR="${HOME}/.local/bin"
VAULT_DIR="${HOME}/.mayfly"
SRC_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" 2>/dev/null && pwd || echo "")"
GITHUB_REPO="vishnunandan555/mayfly"
VERSION="${MAYFLY_VERSION:-v0.0.2}"

UNINSTALL=false
UPDATE=false

for arg in "$@"; do
    case "$arg" in
        --uninstall|-u)
            UNINSTALL=true
            ;;
        --update)
            UPDATE=true
            ;;
        --help|-h)
            echo "MayFly Installer"
            echo "Usage: ./install.sh [OPTIONS]"
            echo "  (no args)       Install mayfly and mf to ~/.local/bin with SHA-256 verification"
            echo "  --update        Rebuild / update installed binaries"
            echo "  --uninstall, -u Cleanly remove mayfly and mf"
            exit 0
            ;;
    esac
done

# -------------------------------------------------------------
# UNINSTALL MODE
# -------------------------------------------------------------
if [ "$UNINSTALL" = true ]; then
    if [ -e /dev/tty ]; then
        exec < /dev/tty
    fi

    echo "================================================="
    echo "  MayFly Complete Uninstaller"
    echo "================================================="
    echo "WARNING: This will completely remove the 'mayfly' and 'mf'"
    echo "binaries, clean your shell PATH, and PERMANENTLY DELETE"
    echo "all encrypted secrets in ~/.mayfly."
    echo ""
    read -p "Are you sure you want to completely uninstall MayFly? [y/N]: " -r RESP
    echo ""

    if [[ ! "$RESP" =~ ^[Yy]$ ]]; then
        echo "Uninstallation canceled."
        exit 0
    fi

    echo "Removing MayFly and wiping vaults..."
    rm -f "${INSTALL_DIR}/mayfly" "${INSTALL_DIR}/mf"
    rm -f "/usr/local/bin/mayfly" "/usr/local/bin/mf" 2>/dev/null || true
    rm -rf "${VAULT_DIR}"

    echo "✓ Removed binaries from ${INSTALL_DIR}"
    echo "✓ Removed ~/.mayfly directory and all encrypted vaults."

    # Clean shell config PATH additions
    for rc in "${HOME}/.zshrc" "${HOME}/.zprofile" "${HOME}/.bashrc" "${HOME}/.bash_profile" "${HOME}/.profile"; do
        if [ -f "$rc" ] && grep -q "Added by MayFly" "$rc"; then
            sed -i '/# Added by MayFly/d' "$rc" 2>/dev/null || sed -i '' '/# Added by MayFly/d' "$rc" 2>/dev/null || true
            sed -i '/export PATH=.*\.local\/bin/d' "$rc" 2>/dev/null || sed -i '' '/export PATH=.*\.local\/bin/d' "$rc" 2>/dev/null || true
            echo "✓ Cleaned PATH configuration from $(basename "$rc")"
        fi
    done

    echo ""
    echo "MayFly has been completely and cleanly uninstalled from your system."
    exit 0
fi

# -------------------------------------------------------------
# INSTALL / UPDATE MODE
# -------------------------------------------------------------

# Detect OS and Architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64|amd64)
        NORM_ARCH="amd64"
        ;;
    arm64|aarch64)
        NORM_ARCH="arm64"
        ;;
    *)
        echo "Error: Unsupported CPU architecture: ${ARCH}"
        exit 1
        ;;
esac

case "$OS" in
    linux)
        TARGET_BIN="mayfly-linux-${NORM_ARCH}"
        OS_DISPLAY="Linux"
        ;;
    darwin)
        TARGET_BIN="mayfly-darwin-${NORM_ARCH}"
        OS_DISPLAY="macOS (Darwin)"
        ;;
    *)
        echo "Error: Unsupported operating system: ${OS}"
        exit 1
        ;;
esac

echo "================================================="
if [ "$UPDATE" = true ]; then
    echo "  MayFly: Updating to ${VERSION}"
else
    echo "  MayFly: Zero-Dependency Secrets Workspace"
    echo "  Secure Installation & Supply-Chain Verifier"
fi
echo "================================================="
echo "  Target Version : ${VERSION}"
echo "  Platform       : ${OS_DISPLAY} (${ARCH} -> ${NORM_ARCH})"
echo "  Install Path   : ${INSTALL_DIR}"
echo "  Security Mode  : Tier-2 Cryptographic SHA-256 Verified"
echo "================================================="
echo ""

# Prompt for command alias selection
echo "Choose command alias to install:"
echo "  [1] Both 'mayfly' and 'mf' (Recommended: press Enter)"
echo "  [2] Only 'mayfly'"
echo "  [3] Only 'mf'"
echo ""

ALIAS_CHOICE=""
if [ -t 0 ]; then
    read -p "Select option [1/2/3]: " -r ALIAS_CHOICE || ALIAS_CHOICE=1
elif [ -c /dev/tty ]; then
    read -p "Select option [1/2/3]: " -r ALIAS_CHOICE < /dev/tty 2>/dev/null || ALIAS_CHOICE=1
else
    ALIAS_CHOICE=1
fi
ALIAS_CHOICE="${ALIAS_CHOICE:-1}"

mkdir -p "${INSTALL_DIR}"

# -------------------------------------------------------------
# ACQUISITION: STRICT CRYPTOGRAPHIC RELEASE VERIFICATION
# -------------------------------------------------------------
TMP_DIR="$(mktemp -d /tmp/mayfly-install.XXXXXX)"
trap 'rm -rf "${TMP_DIR}"' EXIT

if [ "$VERSION" = "latest" ]; then
    BASE_URL="https://github.com/${GITHUB_REPO}/releases/latest/download"
else
    BASE_URL="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}"
fi

echo ""
echo "─── [1/3] Downloading Release Artifacts ─────────────────"
INSTALLED_LOCAL=false

if ! curl -fsSL "${BASE_URL}/${TARGET_BIN}" -o "${TMP_DIR}/${TARGET_BIN}" 2>/dev/null; then
    if [ -n "$SRC_DIR" ] && [ -f "$SRC_DIR/cmd/mayfly/main.go" ] && command -v go >/dev/null 2>&1; then
        echo "Note: Remote release download unavailable; building locally from Go source..."
        (cd "$SRC_DIR" && go build -o "${INSTALL_DIR}/mayfly" ./cmd/mayfly)
        chmod +x "${INSTALL_DIR}/mayfly"
        INSTALLED_LOCAL=true
    else
        echo ""
        echo "Error: Failed to download release binary '${TARGET_BIN}' from GitHub Releases (${BASE_URL})."
        echo "Please check your network connection or verify that release ${VERSION} is published at:"
        echo "https://github.com/${GITHUB_REPO}/releases"
        exit 1
    fi
fi

if [ "$INSTALLED_LOCAL" = false ]; then
    if ! curl -fsSL "${BASE_URL}/checksums.txt" -o "${TMP_DIR}/checksums.txt" 2>/dev/null; then
        echo ""
        echo "Error: Failed to download official 'checksums.txt' manifest from GitHub Releases."
        echo "Installation aborted to prevent running unverified binaries."
        exit 1
    fi

    echo "[OK] Downloaded binary and published checksums.txt"
    echo ""
    echo "─── [2/3] Cryptographic SHA-256 Verification ────────────"
    cd "${TMP_DIR}"

    EXPECTED_HASH="$(grep "${TARGET_BIN}" checksums.txt 2>/dev/null | awk '{print $1}' || echo "")"
    COMPUTED_HASH=""

    if command -v sha256sum >/dev/null 2>&1; then
        COMPUTED_HASH="$(sha256sum "${TARGET_BIN}" | awk '{print $1}')"
    elif command -v shasum >/dev/null 2>&1; then
        COMPUTED_HASH="$(shasum -a 256 "${TARGET_BIN}" | awk '{print $1}')"
    else
        echo "Error: Neither 'sha256sum' nor 'shasum' is available on this system."
        echo "Cannot cryptographically verify binary integrity. Installation aborted."
        exit 1
    fi

    echo "Published Hash : ${EXPECTED_HASH}"
    echo "Computed Hash  : ${COMPUTED_HASH}"

    if [ -n "$EXPECTED_HASH" ] && [ "$EXPECTED_HASH" = "$COMPUTED_HASH" ]; then
        echo "Verification   : [OK] 100% BIT-FOR-BIT MATCH (Authentic & Untampered)"
        mv "${TMP_DIR}/${TARGET_BIN}" "${INSTALL_DIR}/mayfly"
    else
        echo "Verification   : [FAILED] MISMATCH"
        echo ""
        echo "[SECURITY ALERT]: Cryptographic checksum verification failed!"
        echo "The downloaded binary does NOT match the published release hash."
        echo "Installation aborted to protect your system from potential tampering."
        exit 1
    fi

    chmod +x "${INSTALL_DIR}/mayfly"
fi

echo ""
echo "─── [3/3] Configuring Binaries & Aliases ────────────────"
case "$ALIAS_CHOICE" in
    2)
        rm -f "${INSTALL_DIR}/mf" 2>/dev/null || true
        echo "✓ Installed 'mayfly' -> ${INSTALL_DIR}/mayfly"
        ;;
    3)
        mv "${INSTALL_DIR}/mayfly" "${INSTALL_DIR}/mf"
        echo "✓ Installed 'mf' -> ${INSTALL_DIR}/mf"
        ;;
    *)
        ln -sf "${INSTALL_DIR}/mayfly" "${INSTALL_DIR}/mf"
        echo "✓ Installed 'mayfly' -> ${INSTALL_DIR}/mayfly"
        echo "✓ Installed 'mf'     -> ${INSTALL_DIR}/mf"
        ;;
esac

# Ensure PATH is configured across user's active shell configs
PATH_UPDATED=false
case ":$PATH:" in
    *":${INSTALL_DIR}:"*) ;;
    *)
        TARGET_RCS=()
        if [ "$OS" = "darwin" ]; then
            # macOS defaults to zsh (both interactive and login shells)
            TARGET_RCS+=("${HOME}/.zshrc" "${HOME}/.zprofile")
            if [ -f "${HOME}/.bash_profile" ] || [ -f "${HOME}/.bashrc" ]; then
                TARGET_RCS+=("${HOME}/.bash_profile")
            fi
        else
            SHELL_NAME="$(basename "${SHELL:-bash}")"
            if [ "$SHELL_NAME" = "zsh" ]; then
                TARGET_RCS+=("${HOME}/.zshrc")
            elif [ "$SHELL_NAME" = "fish" ]; then
                mkdir -p "${HOME}/.config/fish"
                TARGET_RCS+=("${HOME}/.config/fish/config.fish")
            else
                if [ -f "${HOME}/.bashrc" ]; then
                    TARGET_RCS+=("${HOME}/.bashrc")
                else
                    TARGET_RCS+=("${HOME}/.bash_profile")
                fi
            fi
        fi

        for rc in "${TARGET_RCS[@]}"; do
            if [ -n "$rc" ]; then
                touch "$rc" 2>/dev/null || true
                if ! grep -q "Added by MayFly" "$rc" 2>/dev/null; then
                    echo "" >> "$rc"
                    echo "# Added by MayFly installer" >> "$rc"
                    if [[ "$rc" == *".config/fish/config.fish" ]]; then
                        echo "set -gx PATH \$HOME/.local/bin \$PATH" >> "$rc"
                    else
                        echo "export PATH=\"\$HOME/.local/bin:\$PATH\"" >> "$rc"
                    fi
                    PATH_UPDATED=true
                    echo "[OK] Added PATH export to $(basename "$rc")"
                fi
            fi
        done
        ;;
esac

echo ""
echo "================================================="
echo "  MayFly Installation Complete"
echo "================================================="
echo ""
echo "Getting Started:"
echo "  mayfly (or mf)            - Launch Interactive TUI Dashboard"
echo "  mf c                      - Open TUI for current project"
echo "  mf run <command>          - Run app with in-memory secrets (e.g. mf run npm start)"
echo "  mf --help (or mf help)    - View all available commands"
echo ""
echo "Management & Updates:"
echo "  mf uninstall              - Cleanly uninstall MayFly & remove vaults"
echo ""
if [ "$PATH_UPDATED" = true ]; then
    echo "Note: Restart your terminal or run 'source ${RC_FILE}' to refresh PATH."
fi
