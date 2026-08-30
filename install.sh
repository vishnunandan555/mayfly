#!/usr/bin/env bash
# MayFly — All-in-One Global Installer, Updater, and Uninstaller
# Supports Linux (all distros) and macOS (Darwin Intel & Apple Silicon)

set -e

INSTALL_DIR="${HOME}/.local/bin"
VAULT_DIR="${HOME}/.mayfly"
SRC_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" 2>/dev/null && pwd || echo "")"

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
            echo "  (no args)       Install mayfly and mf to ~/.local/bin"
            echo "  --update        Rebuild and update installed binaries"
            echo "  --uninstall, -u Cleanly remove mayfly and mf"
            exit 0
            ;;
    esac
done

# -------------------------------------------------------------
# UNINSTALL MODE
# -------------------------------------------------------------
if [ "$UNINSTALL" = true ]; then
    # If running piped via curl, reattach stdin to terminal
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
    for rc in "${HOME}/.zshrc" "${HOME}/.bashrc" "${HOME}/.bash_profile" "${HOME}/.profile"; do
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
# Reattach stdin if running piped via curl
if [ -e /dev/tty ]; then
    exec < /dev/tty
fi

echo "================================================="
if [ "$UPDATE" = true ]; then
    echo "  Updating MayFly..."
else
    echo "  Installing MayFly (Zero-Dependency Secrets)..."
fi
echo "================================================="
echo ""

# Prompt for command alias selection
echo "Choose command alias to install:"
echo "  [1] Both 'mayfly' and 'mf' (Default — press Enter)"
echo "  [2] Only 'mayfly'"
echo "  [3] Only 'mf'"
echo ""
read -p "Select option [1/2/3]: " -r ALIAS_CHOICE
ALIAS_CHOICE="${ALIAS_CHOICE:-1}"

mkdir -p "${INSTALL_DIR}"

# Build binary
echo ""
echo "Building binary..."
if [ -n "$SRC_DIR" ] && [ -f "${SRC_DIR}/go.mod" ]; then
    cd "${SRC_DIR}"
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "${INSTALL_DIR}/mayfly" ./cmd/mayfly
else
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "${INSTALL_DIR}/mayfly" ./cmd/mayfly
fi

chmod +x "${INSTALL_DIR}/mayfly"

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

# Ensure PATH is configured
PATH_UPDATED=false
case ":$PATH:" in
    *":${INSTALL_DIR}:"*) ;;
    *)
        # Detect shell configuration
        SHELL_NAME="$(basename "${SHELL:-bash}")"
        RC_FILE=""
        if [ "$SHELL_NAME" = "zsh" ]; then
            RC_FILE="${HOME}/.zshrc"
        elif [ "$SHELL_NAME" = "bash" ]; then
            if [ -f "${HOME}/.bashrc" ]; then
                RC_FILE="${HOME}/.bashrc"
            else
                RC_FILE="${HOME}/.bash_profile"
            fi
        elif [ "$SHELL_NAME" = "fish" ]; then
            mkdir -p "${HOME}/.config/fish"
            RC_FILE="${HOME}/.config/fish/config.fish"
        fi

        if [ -n "$RC_FILE" ]; then
            echo ""
            echo "Note: '${INSTALL_DIR}' is not currently in your system PATH."
            read -p "Add '${INSTALL_DIR}' to $(basename "$RC_FILE")? [Y/n]: " -r ADD_PATH_RESP
            ADD_PATH_RESP="${ADD_PATH_RESP:-y}"

            if [[ "$ADD_PATH_RESP" =~ ^[Yy]$ ]]; then
                if ! grep -q "Added by MayFly" "$RC_FILE" 2>/dev/null; then
                    echo "" >> "$RC_FILE"
                    echo "# Added by MayFly installer" >> "$RC_FILE"
                    echo "export PATH=\"\$HOME/.local/bin:\$PATH\"" >> "$RC_FILE"
                    PATH_UPDATED=true
                    echo "✓ Added PATH export to $(basename "$RC_FILE")"
                fi
            fi
        fi
        ;;
esac

echo ""
echo "================================================="
echo "  🎉 MayFly Installation Complete!"
echo "================================================="
echo ""
echo "Getting Started:"
echo "  mayfly (or mf)            - Launch Global TUI Dashboard"
echo "  mf c                      - Open TUI for current project"
echo "  mf run <command>          - Run app with in-memory secrets (e.g. mf run npm start)"
echo "  mf --help (or mf help)    - View all available commands"
echo ""
echo "Management & Updates:"
echo "  mf uninstall              - Cleanly uninstall MayFly & remove binaries"
echo "  curl -fsSL https://raw.githubusercontent.com/vishnunandan555/mayfly/main/install.sh | bash -s -- --update"
echo "                            - Update / rebuild latest MayFly release"
echo ""
if [ "$PATH_UPDATED" = true ]; then
    echo "Note: Restart your terminal or run 'source ${RC_FILE}' to refresh PATH."
fi
