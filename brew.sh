# Make sure we’re using the latest Homebrew.
# brew update
# Upgrade any already-installed formulae.
# brew upgrade

# Python
brew install python
brew install python-tk  # useful if anything is using tk such as pandas_profiling

# Install a modern version of Bash.
brew install bash
brew install bash-completion2

# Install apps
brew install --cask atom

# zsh sundries
brew install romkatv/powerlevel10k/powerlevel10k

# Install useful tools
brew install bat
brew install xsv  # CSV CLI tools written in Rust
brew install csvkit
brew install neovim
brew install git
brew install git-lfs
brew install git-delta
brew install gh  # GitHub CLI
brew install exa  # modern ls replacement
# brew install csvkit

 # fuzzy command line finder https://github.com/junegunn/fzf#usage
brew install fzf
$(brew --prefix)/opt/fzf/install

# Remove outdated versions from the cellar.
brew cleanup
