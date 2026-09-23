# ── Taps ─────────────────────────────────────────────────────────────────────
tap "hashicorp/tap"
tap "supabase/tap"

# ── Languages & runtimes ────────────────────────────────────────────────────
brew "python"
brew "python-tk@3.14"                     # Tk bindings for Python
brew "uv"                                 # Fast Python package manager
brew "zsh"
brew "bash"
# brew "go"
brew "rust"
brew "fnm"                                # Fast Node version manager
brew "pnpm"
# brew "node"
# brew "deno"

# ── Shell ────────────────────────────────────────────────────────────────────
brew "powerlevel10k"
brew "direnv"                             # Per-directory env vars

# ── CLI tools ────────────────────────────────────────────────────────────────
brew "git"
brew "git-lfs"
brew "git-delta"                          # Beautiful git diffs
brew "git-filter-repo"                    # Rewrite git history
brew "gh"                                 # GitHub CLI
brew "glab"                               # GitLab CLI
brew "fzf"                                # Fuzzy finder (shell integration in setup.sh)
brew "ripgrep"                            # rg — fast grep
brew "bat"                                # cat with syntax highlighting
brew "eza"                                # Modern ls replacement
brew "jq"                                 # JSON processor
brew "neovim"
brew "tmux"
brew "wget"
brew "watchexec"                          # File watcher
brew "shellcheck"                         # Shell script linter
brew "moreutils"                          # sponge, ts, vidir, etc.
brew "csvkit"                             # CSV tools
brew "s3cmd"                              # S3 command-line tool
brew "scc"                                # Code counter
brew "pandoc"                             # Document conversion
brew "poppler"                            # pdftotext and friends
brew "weasyprint"                         # HTML to PDF
brew "pngquant"                           # PNG compression
brew "terminal-notifier"                  # macOS notifications from the CLI
brew "cliclick"                           # Mouse/keyboard automation
brew "ical-buddy"                         # Read macOS calendar
brew "gogcli"                             # Google Workspace CLI
brew "herdr"                              # Agent multiplexer
# brew "coreutils"                        # GNU core utilities
# brew "htop"
# brew "btop"
# brew "glow"                             # Markdown renderer for terminal
# brew "broot"                            # Interactive directory navigator
# brew "xsv"                              # CSV tools (Rust)

# ── Dev & security ───────────────────────────────────────────────────────────
brew "ruff"                               # Python linter
brew "actionlint"                         # GitHub Actions linter
brew "gitleaks"                           # Secret scanning
brew "trufflehog"                         # Secret scanning
brew "cmake"
brew "xcodegen"

# ── Cloud & infrastructure ───────────────────────────────────────────────────
brew "supabase/tap/supabase", trusted: true
brew "hashicorp/tap/terraform", trusted: true
brew "cloud-sql-proxy"
brew "cloudflare-wrangler"
brew "crane"                              # Container registry tool
brew "doctl"                              # DigitalOcean CLI
brew "flyctl"                             # Fly.io CLI
brew "neonctl"                            # Neon CLI
brew "sentry-cli"
brew "awscli"
# brew "kubernetes-cli"
# brew "helm"
# brew "pulumi"
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
# brew "yt-dlp"

# ── Dev extras ───────────────────────────────────────────────────────────────
# brew "pipx"
brew "yarn"

# ── Cask apps ────────────────────────────────────────────────────────────────

# Terminals & editors
cask "ghostty"
cask "iterm2"
cask "pulsar"
cask "jetbrains-toolbox"
cask "opencode-desktop"
# cask "visual-studio-code"
# cask "cursor"
# cask "warp"

# Browsers
cask "google-chrome"
cask "firefox"
# cask "arc"

# Communication
cask "slack"
cask "microsoft-teams"
cask "microsoft-outlook"
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
# cask "lastpass"

# Productivity
cask "alfred"
cask "asana"
cask "microsoft-excel"
cask "deckset"                            # Markdown presentations
cask "obsidian"
# cask "bear"
# cask "joplin"
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
