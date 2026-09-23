# CLAUDE.md

## Project overview

Personal dotfiles for macOS. Manages shell config (zsh + oh-my-zsh + Powerlevel10k), Homebrew packages, aliases, exports, and a journaling utility.

## Repository structure

```
Brewfile              # Homebrew packages, casks, and taps
setup.sh              # Full bootstrap script (safe to re-run)
journal.sh            # CLI journaling tool (aliased as jj)
neovim_install.sh     # Neovim setup
bin/                  # Scripts on PATH (git-prune-merged: delete merged branches + worktrees)
shell/
  .alias              # Shell aliases and functions
  .exports            # Non-secret environment variables
  .secrets            # Secret env vars (gitignored)
  .gitconfig          # Git configuration (symlinked to ~/)
  .gitconfig-*.example # Git identity templates
zsh/
  .zshrc              # Main zsh config (symlinked to ~/)
  .p10k.zsh           # Powerlevel10k theme config
```

## Key conventions

- Shell: zsh with oh-my-zsh and Powerlevel10k prompt
- `setup.sh` is the single entry point for bootstrapping a new Mac (stages 0-6)
- Configs are symlinked into `~/` by `setup.sh` -- never copy, always symlink
- Secrets go in `shell/.secrets` (gitignored), never commit credentials
- Commented-out lines in `Brewfile` are intentional -- they're optional packages that can be enabled as needed
- `DOTFILE_DIR` is set to `$HOME/repos/dotfiles/` and used for sourcing modules

## Editing guidelines

- When adding new tools: add the brew formula/cask to `Brewfile`, any aliases to `shell/.alias`, any env vars to `shell/.exports`
- When adding shell integrations: add to the "Tool integrations" section of `zsh/.zshrc`
- Keep `setup.sh` idempotent -- guard new steps with existence checks
- Preserve the section-header comment style (`# ── Section ──`) used throughout
