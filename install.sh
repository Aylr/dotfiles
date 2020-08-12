#!/bin/bash

# Set up symlinks
ln -s bash/.bash_profile ~/.bash_profile
ln -s bash/.gitconfig ~/.gitconfig


#######################
# Neovim
#######################

# Config symlinks
mkdir ~/.config/nvim || echo 'nvim config directory exists'
mkdir ~/.config/nvim/plugged || echo 'nvim plugged directory exists'
ln -s "$(pwd)/nvim/autoload" ~/.config/nvim/autoload
ln -s "$(pwd)/nvim/init.vim" ~/.config/nvim/init.vim

# Plugins. Run twice so themes are all set up.
nvim --headless +PlugInstall +qa
nvim --headless +PlugInstall +qa

