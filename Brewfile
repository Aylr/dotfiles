# ── Taps ─────────────────────────────────────────────────────────────────────
tap "supabase/tap"

# ── Languages & runtimes ────────────────────────────────────────────────────
brew "python"
brew "python-tk@3.14"                     # Tk bindings for Python
brew "uv"                                 # Fast Python package manager
brew "zsh"
brew "bash"
brew "rust"

# ── Shell ────────────────────────────────────────────────────────────────────
brew "powerlevel10k"

# ── CLI tools ────────────────────────────────────────────────────────────────
brew "git"
brew "git-lfs"
brew "git-delta"                          # Beautiful git diffs
brew "gh"                                 # GitHub CLI
brew "fzf"                                # Fuzzy finder (shell integration in setup.sh)
brew "ripgrep"                            # rg — fast grep
brew "bat"                                # cat with syntax highlighting
brew "eza"                                # Modern ls replacement
brew "jq"                                 # JSON processor
brew "neovim"
brew "tmux"
brew "wget"
brew "watchexec"                          # File watcher
brew "csvkit"                             # CSV tools
brew "s3cmd"                              # S3 command-line tool
brew "scc"                                # Code counter
brew "pandoc"                             # Document conversion
brew "pngquant"                           # PNG compression
brew "terminal-notifier"                  # macOS notifications from the CLI
brew "cliclick"                           # Mouse/keyboard automation
brew "ical-buddy"                         # Read macOS calendar
brew "gogcli"                             # Google Workspace CLI
brew "herdr"                              # Agent multiplexer

# ── Dev & security ───────────────────────────────────────────────────────────
brew "ruff"                               # Python linter
brew "actionlint"                         # GitHub Actions linter
brew "cmake"
brew "xcodegen"

# ── Cloud & infrastructure ───────────────────────────────────────────────────
brew "supabase/tap/supabase", trusted: true
brew "cloudflare-wrangler"
brew "sentry-cli"
brew "awscli"
# brew "colima"

# ── Databases ────────────────────────────────────────────────────────────────
brew "postgresql@15"
brew "libpq"                              # Postgres client libs (keg-only)
brew "duckdb"
brew "redis", restart_service: :changed

# ── AI / ML ──────────────────────────────────────────────────────────────────
brew "ollama", restart_service: :changed

# ── Media ────────────────────────────────────────────────────────────────────
brew "ffmpeg"

# ── Dev extras ───────────────────────────────────────────────────────────────
brew "yarn"

# ── Cask apps ────────────────────────────────────────────────────────────────

# Terminals & editors
cask "ghostty"
cask "iterm2"
cask "pulsar"
cask "jetbrains-toolbox"

# Browsers
cask "google-chrome"
cask "firefox"

# Communication
cask "slack"
cask "zoom"
cask "telegram"

# Dev tools
cask "tailscale-app"
cask "gcloud-cli"
cask "session-manager-plugin"             # AWS SSM
# cask "postgres-app"                     # Postgres.app
# cask "docker"
# cask "tableplus"
# cask "postman"
cask "ngrok"
cask "cyberduck"

# Passwords
cask "1password"
cask "1password-cli"

# Productivity
cask "alfred"
cask "asana"
cask "microsoft-excel"
cask "deckset"                            # Markdown presentations
cask "obsidian"
# cask "bear"
# cask "xmind"
# cask "loom"

# Creative & media
cask "spotify"
cask "inkscape"
cask "bambu-studio"                       # 3D printer slicer
cask "freecad"
# cask "figma"
# cask "blender"
# cask "vlc"
# cask "obs"
# cask "gimp"
# cask "openscad"
# cask "gifox"
# cask "gifski"
# cask "kicad"
# cask "lightburn"
# cask "autodesk-fusion"

# System utilities
cask "flux-app"
cask "appcleaner"
# cask "karabiner-elements"
cask "caffeine"
cask "swish"
# cask "sound-control"
# cask "1blocker"
# cask "macfuse"
# cask "virtualbox"

# Hardware & maker
# cask "raspberry-pi-imager"
# cask "keymapp"                          # ZSA keyboard configurator
# cask "balenaetcher"
# cask "qmk-toolbox"

# Voice & dictation
cask "opensuperwhisper"
# cask "aqua-voice"
# cask "wispr-flow"
