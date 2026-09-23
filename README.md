# dotfiles

Portable macOS config: zsh + oh-my-zsh + Powerlevel10k, Homebrew packages, git config, aliases, and a journaling utility.

Inspired by [Getting Started With Dotfiles](https://medium.com/@webprolific/getting-started-with-dotfiles-43c3602fd789).

## New Mac setup

```zsh
mkdir -p ~/repos && git clone git@github.com:Aylr/dotfiles.git ~/repos/dotfiles
cd ~/repos/dotfiles && ./setup.sh
```

The repo must live at `~/repos/dotfiles`. `setup.sh` is safe to re-run. It installs Xcode CLT, Homebrew, everything in the `Brewfile`, oh-my-zsh, and Powerlevel10k fonts. It also sets brew's zsh as the login shell and symlinks the configs into `~/`, backing up anything already there.

Then, by hand:

1. Fill in `~/.gitconfig-professional` and `~/.gitconfig-personal` (created from the `.example` templates).
2. Copy `shell/.secrets` over from the old Mac.
3. `gh auth login`
4. `p10k configure` if the prompt looks off.

## Layout

| Path | What |
|------|------|
| `Brewfile` | Homebrew formulae and casks. Commented lines are optional extras. |
| `setup.sh` | Bootstrap script |
| `zsh/.zshrc`, `zsh/.p10k.zsh` | Shell and prompt, symlinked to `~/` |
| `shell/.alias` | Aliases and functions |
| `shell/.exports` | Non-secret environment variables |
| `shell/.secrets` | Secret environment variables (gitignored) |
| `shell/.gitconfig` | Git config, symlinked to `~/`; picks identity by repo path |
| `bin/` | Scripts on `PATH`, e.g. `git prune-merged` |
| `journal.sh` | Journaling CLI, aliased as `jj` |

## Keeping the Brewfile current

```zsh
brew bundle check --verbose   # what's in the Brewfile but not installed
brew bundle cleanup           # what's installed but not in the Brewfile (dry run)
```

## Journal

`jj` views today's entry, `jj -e` (or `j`) edits it in `$EDITOR`.

```
journal.sh                              View today's entry
journal.sh -e                           Edit today's entry
journal.sh "TIL something neat"         Add a note to today's entry
journal.sh "TIL something neat" -e      Add a note and edit
journal.sh -d 2020-01-01                View the entry from 1/1/20
cat file.txt | journal.sh               Pipe file.txt into today's entry
```
