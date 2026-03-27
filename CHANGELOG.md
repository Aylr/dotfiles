# Changelog

## [0.4.0] - 2026-03-27

### Changed

* Replaced `brew.sh` with a declarative `Brewfile` (`brew bundle`)
* Renamed `bash/` to `shell/` since configs are shared with zsh
* Stripped `.zshrc` down from ~160 lines to ~50 (removed oh-my-zsh boilerplate)
* Stripped `.exports` down to essentials (removed old Python 3.7/3.9/3.11, Spark, Scala)
* All paths are now Apple Silicon-aware via `BREW_PREFIX` detection
* Optional tool sources (broot, cargo, NVM, etc.) are guarded with existence checks
* `setup.sh` Stage 2 now uses `brew bundle` instead of `bash brew.sh`
* `setup.sh` post-install checklist trimmed to only items that still need manual action
* `gh` credential helper uses PATH lookup instead of hardcoded `/usr/local/bin/gh`
* Python PATH uses version-agnostic `$BREW_PREFIX/opt/python/libexec/bin`

### Added

* `Brewfile` for declarative Homebrew package management
* `.gitconfig-professional.example` and `.gitconfig-personal.example` templates
* `setup.sh` auto-copies gitconfig identity examples on first run

### Removed

* `brew.sh` (superseded by Brewfile)
* `install.sh` (superseded by setup.sh)
* `.bash_profile` symlink from setup.sh (fully on zsh now)
* Dead aliases: `pandas_profiling`, `sclack`, Travis CI debug function
* Stale Google Cloud SDK path (`~/Downloads/`), Node 16 path
* Commented-out Python/Spark/Scala configs from `.exports`

### Fixed

* `/usr/loca/bin` typo in `.zshrc`
* `alias p'pulumi'` syntax error
* `pp` alias conflict (pandas_profiling vs pulumi preview)
* Duplicate tailscale install in brew.sh (formula + cask)
* Duplicate fzf interactive install (brew.sh + setup.sh)
* Hardcoded `/Users/taylor/` in `jjw` function

## [0.3.0] - 2025-03-25

### Added

* `setup.sh` — 6-stage bootstrap script for new Mac setup
* Powerlevel10k font auto-installation
* Idempotent symlink management with backup

## [Unreleased from before]

### Added

* neovim install, plugins, and configs
* Default git branch name uses more inclusive language
* lots of new aliases for git, docker, and more

## [0.2.0] - 2020-06-20

### Added

* A new journaling command for lightning fast journaling
* zsh compatibility

### Changed

* fiddled with some git aliases
