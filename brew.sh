# Make sure we’re using the latest Homebrew.
brew update
# Upgrade any already-installed formulae.
brew upgrade

# Install a modern version of Bash.
brew install bash
brew install bash-completion2

# Install useful tools
brew install neovim
brew install git
brew install git-lfs
brew install eynx
brew install gh  # GitHub CLI
brew install csvkit

 # fuzzy command line finder https://github.com/junegunn/fzf#usage
brew install fzf 
$(brew --prefix)/opt/fzf/install

# Remove outdated versions from the cellar.
brew cleanup

