# Make sure we're using the latest Homebrew.
brew update
# Upgrade any already-installed formulae.
brew upgrade

# Python
brew install python
brew install python-tk                     # useful if anything is using tk such as pandas_profiling

# Install a modern version of Bash.
brew install bash
brew install bash-completion2

# Install apps
brew install --cask pulsar
brew install --cask alfred
brew install --cask spotify
brew install --cask ghostty
brew install --cask google-chrome
brew install --cask firefox
brew install --cask zoom
brew install --cask flux-app
brew install --cask inkscape
brew install --cask microsoft-teams

# zsh sundries
brew install romkatv/powerlevel10k/powerlevel10k

# Install useful tools
brew install ripgrep                       # rg fast grep tool
brew install bat
#brew install xsv                         # CSV CLI tools written in Rust
brew install csvkit
brew install neovim
brew install git
brew install git-lfs
brew install git-delta
brew install gh                            # GitHub CLI
brew install eza                           # modern ls replacement
brew install watchexec                     # file watcher: https://github.com/watchexec/watchexec
brew install jq

# fuzzy command line finder https://github.com/junegunn/fzf#usage
brew install fzf
$(brew --prefix)/opt/fzf/install

# Dev tools
brew install tmux
brew install supabase/tap/supabase
brew install tailscale
brew install --cask iterm2
brew install --cask tailscale

# Languages & runtimes
#brew install go                          # Go programming language
#brew install rust                        # Rust programming language
#brew install node                        # Node.js JavaScript runtime
#brew install deno                        # Deno JavaScript/TypeScript runtime
#brew install zsh                         # Z shell

# Shell essentials
brew install wget                        # File downloader
#brew install coreutils                   # GNU core utilities
#brew install htop                        # Interactive process viewer
#brew install btop                        # Resource monitor (prettier htop)
#brew install glow                        # Markdown renderer for terminal
#brew install broot                       # Interactive directory navigator

# Cloud & infrastructure
#brew install awscli                      # AWS command-line interface
#brew install kubernetes-cli              # kubectl for managing k8s clusters
#brew install helm                        # Kubernetes package manager
#brew install pulumi                      # Infrastructure as code
#brew install colima                      # Container runtime (Docker alternative)
#brew install --cask google-cloud-sdk     # Google Cloud CLI and tools

# Databases
#brew install duckdb                      # In-process analytical database
#brew install redis                       # In-memory data store

# AI / ML
#brew install ollama                      # Run large language models locally

# Media
#brew install ffmpeg                      # Audio/video processing
#brew install yt-dlp                      # YouTube and video downloader

# Data & observability
brew install s3cmd                       # S3 command-line tool
#brew install rill                        # Local data exploration tool
#brew install sentry-cli                  # Sentry error tracking CLI

# Editors & IDEs
#brew install --cask visual-studio-code   # VS Code editor
#brew install --cask pycharm              # Python IDE by JetBrains

# Dev extras
#brew install pipx                        # Install Python CLI tools in isolated envs
#brew install yarn                        # JavaScript package manager
#brew install --cask postman              # API client and testing
#brew install --cask ngrok                # Expose local servers to the internet

# Creative & design
#brew install --cask gimp                 # Open-source image editor
#brew install --cask openscad             # Programmable 3D CAD modeler

# System & hardware
#brew install --cask karabiner-elements   # Keyboard remapper for macOS
#brew install --cask macfuse              # Filesystem extension framework
#brew install --cask virtualbox           # Virtual machine manager
#brew install --cask qmk-toolbox          # QMK keyboard firmware flasher

# Productivity & notes
#brew install --cask obsidian             # Note-taking with markdown and graph view
#brew install --cask bear                 # Focused writing and note-taking
#brew install --cask joplin               # Open-source note-taking with sync
#brew install --cask xmind                # Mind mapping tool
#brew install --cask loom                 # Video messaging and screen recording
brew install --cask deckset              # Markdown-based presentation tool

# Dev apps
brew install --cask claude               # Claude desktop app
#brew install --cask cursor               # AI-powered code editor
#brew install --cask docker               # Docker Desktop
#brew install --cask warp                 # Modern terminal with AI features
#brew install --cask tableplus            # Database GUI client
brew install --cask postgres-unofficial  # Postgres.app — local PostgreSQL
brew install --cask jetbrains-toolbox    # Manage all JetBrains IDEs
#brew install --cask cyberduck            # FTP/S3/cloud file transfer

# Browsers
#brew install --cask arc                  # Arc browser

# Communication
brew install --cask slack                # Team messaging
#brew install --cask telegram             # Messaging app

# Creative & media
#brew install --cask figma                # Collaborative design tool
#brew install --cask blender              # 3D creation suite
#brew install --cask vlc                  # Media player
#brew install --cask obs                  # Streaming and screen recording
#brew install --cask gifox                # GIF recorder for macOS
#brew install --cask gifski               # High-quality GIF converter

# System utilities
brew install --cask appcleaner           # Cleanly uninstall apps
#brew install --cask 1blocker             # Ad and tracker blocker
#brew install --cask caffeine             # Prevent Mac from sleeping
#brew install --cask swish                # Trackpad gesture shortcuts
#brew install --cask sound-control        # Per-app audio volume control
#brew install --cask aqua-voice           # Voice control app
#brew install --cask wispr-flow           # AI voice dictation

# Hardware & maker tools
#brew install --cask kicad                # PCB and schematic design
#brew install --cask raspberry-pi-imager  # Flash Raspberry Pi SD cards
#brew install --cask keymapp              # ZSA keyboard configurator
#brew install --cask balenaetcher         # Flash OS images to SD cards/USB
brew install --cask bambu-studio         # Bambu Lab 3D printer slicer
#brew install --cask lightburn            # Laser cutter/engraver software
#brew install --cask autodesk-fusion      # 3D CAD/CAM design tool

# Remove outdated versions from the cellar.
brew cleanup
