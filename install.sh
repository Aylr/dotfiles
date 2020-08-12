#!/bin/bash

# Set up symlinks
ln -s bash/.bash_profile ~/.bash_profile
ln -s bash/.gitconfig ~/.gitconfig



# Install command-line tools using Homebrew. h/t https://github.com/mathiasbynens/dotfiles

# Make sure we’re using the latest Homebrew.
brew update
# Upgrade any already-installed formulae.
brew upgrade

# Save Homebrew’s installed location.
BREW_PREFIX=$(brew --prefix)

# Install a modern version of Bash.
brew install bash
brew install bash-completion2

# Install useful tools
brew install neovim
brew install git
brew install git-lfs
brew install lynx

# Remove outdated versions from the cellar.
brew cleanup
