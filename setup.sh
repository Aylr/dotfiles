#!/bin/bash
#
# setup.sh — bootstrap a new Mac from this dotfiles repo
#
# Usage:
#   bash setup.sh          # full setup
#   bash setup.sh --help   # show usage
#
# Safe to re-run. Existing files are backed up, not overwritten.

set -euo pipefail

DOTFILES_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ── Colors & helpers ────────────────────────────────────────────────────────

BOLD='\033[1m'
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

info()    { echo -e "${BLUE}▸${NC} $*"; }
success() { echo -e "${GREEN}✓${NC} $*"; }
warn()    { echo -e "${YELLOW}!${NC} $*"; }
error()   { echo -e "${RED}✗${NC} $*"; exit 1; }
header()  { echo -e "\n${BOLD}── $* ──${NC}"; }

# ── Usage ───────────────────────────────────────────────────────────────────

if [[ "${1:-}" == "--help" ]]; then
    echo "Usage: bash setup.sh"
    echo ""
    echo "Stages:"
    echo "  0  Prerequisites   (Xcode CLT)"
    echo "  1  Homebrew        (install + update)"
    echo "  2  Packages        (brew.sh)"
    echo "  3  Shell           (zsh default, oh-my-zsh, p10k fonts)"
    echo "  4  Symlinks        (shell configs, gitconfig)"
    echo "  5  fzf             (shell integration)"
    echo "  6  Neovim          (skipped if no nvim/ dir in dotfiles)"
    exit 0
fi

# ── Stage 0: Prerequisites ───────────────────────────────────────────────────

header "Stage 0: Prerequisites"

[[ "$(uname)" == "Darwin" ]] || error "This script is macOS only."
success "macOS detected."

if ! xcode-select -p &>/dev/null; then
    warn "Xcode Command Line Tools not found — requesting install..."
    xcode-select --install
    echo ""
    warn "Finish the Xcode CLT install dialog, then re-run this script."
    exit 0
fi
success "Xcode Command Line Tools installed."

# ── Stage 1: Homebrew ────────────────────────────────────────────────────────

header "Stage 1: Homebrew"

if ! command -v brew &>/dev/null; then
    info "Installing Homebrew..."
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

    # Add brew to PATH immediately (Apple Silicon puts it in /opt/homebrew)
    if [[ -f /opt/homebrew/bin/brew ]]; then
        eval "$(/opt/homebrew/bin/brew shellenv)"
    elif [[ -f /usr/local/bin/brew ]]; then
        eval "$(/usr/local/bin/brew shellenv)"
    fi
    success "Homebrew installed."
else
    success "Homebrew already installed."
fi

# ── Stage 2: Packages ────────────────────────────────────────────────────────

header "Stage 2: Packages (brew.sh)"

bash "$DOTFILES_DIR/brew.sh"
success "Packages installed."

# ── Stage 3: Shell ───────────────────────────────────────────────────────────

header "Stage 3: Shell"

# Set brew's zsh as the default shell
BREW_ZSH="$(brew --prefix)/bin/zsh"
if [[ "$SHELL" != "$BREW_ZSH" ]]; then
    if ! grep -qF "$BREW_ZSH" /etc/shells; then
        info "Adding $BREW_ZSH to /etc/shells..."
        echo "$BREW_ZSH" | sudo tee -a /etc/shells > /dev/null
    fi
    info "Changing default shell to $BREW_ZSH..."
    chsh -s "$BREW_ZSH"
    success "Default shell set to brew zsh."
else
    success "Default shell is already brew zsh."
fi

# oh-my-zsh
if [[ ! -d "$HOME/.oh-my-zsh" ]]; then
    info "Installing oh-my-zsh..."
    RUNZSH=no CHSH=no sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"
    success "oh-my-zsh installed."
else
    success "oh-my-zsh already installed."
fi

# Powerlevel10k fonts
info "Installing Powerlevel10k fonts..."
FONTS_DIR="$HOME/Library/Fonts"
P10K_FONTS=(
    "MesloLGS%20NF%20Regular.ttf"
    "MesloLGS%20NF%20Bold.ttf"
    "MesloLGS%20NF%20Italic.ttf"
    "MesloLGS%20NF%20Bold%20Italic.ttf"
)
for encoded in "${P10K_FONTS[@]}"; do
    filename="${encoded//%20/ }"
    if [[ ! -f "$FONTS_DIR/$filename" ]]; then
        curl -fsSL "https://raw.githubusercontent.com/romkatv/powerlevel10k-media/master/$encoded" \
            -o "$FONTS_DIR/$filename"
    fi
done
success "Powerlevel10k fonts ready."

# ── Stage 4: Symlinks ────────────────────────────────────────────────────────

header "Stage 4: Symlinks"

symlink() {
    local src="$1"
    local dst="$2"

    if [[ -L "$dst" ]]; then
        if [[ "$(readlink "$dst")" == "$src" ]]; then
            warn "Symlink already correct: $dst"
        else
            warn "Symlink exists but points elsewhere: $dst → $(readlink "$dst")"
            warn "  Skipping. Remove manually if you want to update it."
        fi
        return
    fi

    if [[ -f "$dst" || -d "$dst" ]]; then
        local backup="${dst}.backup_$(date +%Y%m%d_%H%M%S)"
        warn "Backing up $dst → $backup"
        mv "$dst" "$backup"
    fi

    ln -s "$src" "$dst"
    success "Linked $dst → $src"
}

symlink "$DOTFILES_DIR/zsh/.zshrc"         "$HOME/.zshrc"
symlink "$DOTFILES_DIR/zsh/.p10k.zsh"      "$HOME/.p10k.zsh"
symlink "$DOTFILES_DIR/bash/.bash_profile"  "$HOME/.bash_profile"
symlink "$DOTFILES_DIR/bash/.gitconfig"     "$HOME/.gitconfig"

# ── Stage 5: fzf shell integration ──────────────────────────────────────────

header "Stage 5: fzf"

if command -v fzf &>/dev/null; then
    "$(brew --prefix)/opt/fzf/install" --all --no-update-rc --no-bash --no-fish 2>/dev/null || true
    success "fzf shell integration installed."
else
    warn "fzf not found — skipping. Run brew.sh first."
fi

# ── Stage 6: Neovim (optional) ───────────────────────────────────────────────

header "Stage 6: Neovim"

if [[ -d "$DOTFILES_DIR/nvim" ]]; then
    info "Found nvim config, running neovim_install.sh..."
    bash "$DOTFILES_DIR/neovim_install.sh"
    success "Neovim configured."
else
    warn "No nvim/ directory found in dotfiles — skipping Neovim setup."
fi

# ── Done: Manual checklist ───────────────────────────────────────────────────

echo ""
echo -e "${GREEN}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}${BOLD}  All done! A few things still need manual attention:${NC}"
echo -e "${GREEN}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "  ${BOLD}1. Git identity${NC}"
echo    "     Edit bash/.gitconfig and set your name + email under [user]"
echo ""
echo -e "  ${BOLD}2. Secrets file${NC}"
echo    "     Create ~/.secrets with your API keys and tokens"
echo    "     (see bash/.secrets if you have a backup)"
echo ""
echo -e "  ${BOLD}3. GitHub CLI auth${NC}"
echo    "     gh auth login"
echo ""
echo -e "  ${BOLD}4. Update DOTFILE_DIR in bash/.bash_profile${NC}"
echo    "     Set it to: $DOTFILES_DIR"
echo ""
echo -e "  ${BOLD}5. Python PATH in bash/.exports${NC}"
echo    "     Update the Python version path — check with:"
echo    "     brew --prefix python@3.13"
echo ""
echo -e "  ${BOLD}6. iTerm2 color scheme${NC}"
echo    "     Preferences → Profiles → Colors → Color Presets → Import"
echo    "     File: $DOTFILES_DIR/Adventure.itermcolors"
echo ""
echo -e "  ${BOLD}7. Configure the p10k prompt${NC}"
echo    "     p10k configure"
echo ""
echo -e "  ${BOLD}8. Restart your terminal.${NC}"
echo ""
