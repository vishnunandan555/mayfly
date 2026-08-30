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
    echo "Uninstalling MayFly..."
    
    rm -f "${INSTALL_DIR}/mayfly"
    rm -f "${INSTALL_DIR}/mf"
    rm -f "/usr/local/bin/mayfly" 2>/dev/null || true
    rm -f "/usr/local/bin/mf" 2>/dev/null || true

    echo "✓ Removed 'mayfly' and 'mf' binaries from ${INSTALL_DIR}"

    # Clean shell config PATH additions
    for rc in "${HOME}/.zshrc" "${HOME}/.bashrc" "${HOME}/.bash_profile" "${HOME}/.profile"; do
        if [ -f "$rc" ] && grep -q "Added by MayFly" "$rc"; then
            # Cleanly remove the MayFly PATH block
            sed -i '/# Added by MayFly/d' "$rc" 2>/dev/null || sed -i '' '/# Added by MayFly/d' "$rc" 2>/dev/null || true
            sed -i '/export PATH=.*\.local\/bin/d' "$rc" 2>/dev/null || sed -i '' '/export PATH=.*\.local\/bin/d' "$rc" 2>/dev/null || true
            echo "✓ Cleaned PATH configuration from $(basename "$rc")"
        fi
    done

    if [ -d "${VAULT_DIR}" ]; then
        read -p "Do you want to permanently delete encrypted secrets in ~/.mayfly? [y/N]: " -r RESP
        if [[ "$RESP" =~ ^[Yy]$ ]]; then
            rm -rf "${VAULT_DIR}"
            echo "✓ Removed ~/.mayfly directory."
        else
            echo "Kept ~/.mayfly intact for future use."
        fi
    fi

    echo ""
    echo "MayFly has been completely and cleanly uninstalled from your system."
    exit 0
fi

# -------------------------------------------------------------
# INSTALL / UPDATE MODE
# -------------------------------------------------------------
echo "================================================="
if [ "$UPDATE" = true ]; then
    echo "  Updating MayFly..."
else
    echo "  Installing MayFly (Zero-Dependency Secrets)..."
fi
echo "================================================="

mkdir -p "${INSTALL_DIR}"

# Build binary
if [ -n "$SRC_DIR" ] && [ -f "${SRC_DIR}/go.mod" ]; then
    echo "Building from source..."
    cd "${SRC_DIR}"
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "${INSTALL_DIR}/mayfly" ./cmd/mayfly
else
    echo "Building mayfly binary..."
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "${INSTALL_DIR}/mayfly" ./cmd/mayfly
fi

chmod +x "${INSTALL_DIR}/mayfly"

# Create 'mf' alias symlink
ln -sf "${INSTALL_DIR}/mayfly" "${INSTALL_DIR}/mf"

echo "✓ Installed 'mayfly' -> ${INSTALL_DIR}/mayfly"
echo "✓ Installed 'mf'     -> ${INSTALL_DIR}/mf"

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

        if [ -n "$RC_FILE" ] && [ -f "$RC_FILE" ]; then
            if ! grep -q "Added by MayFly" "$RC_FILE"; then
                echo "" >> "$RC_FILE"
                echo "# Added by MayFly installer" >> "$RC_FILE"
                echo "export PATH=\"\$HOME/.local/bin:\$PATH\"" >> "$RC_FILE"
                PATH_UPDATED=true
            fi
        fi
        ;;
esac

echo ""
echo "================================================="
echo "  🎉 Installation Complete!"
echo "================================================="
echo ""
echo "You can now run either:"
echo "  mayfly        - Launch Global TUI (Project Grid)"
echo "  mf            - Short alias"
echo "  mf c          - Open TUI for current project"
echo "  mf run <cmd>  - Run app with in-memory secrets"
echo "  mf help       - View all commands"
echo ""
if [ "$PATH_UPDATED" = true ]; then
    echo "Note: Restart your terminal or run 'source ${RC_FILE}' to refresh PATH."
fi
