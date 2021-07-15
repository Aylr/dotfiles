#!/bin/bash

# install oh-my-zsh
# sh -c "$(curl -fsSL https://raw.github.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"

# Set up symlinks
ln -s "$(pwd)"/zsh/.zshrc ~/.zshrc
ln -s "$(pwd)"/bash/.bash_profile ~/.bash_profile
ln -s "$(pwd)"/bash/.gitconfig ~/.gitconfig


#######################
# Neovim
#######################

# Config symlinks
mkdir ~/.config/nvim || echo 'nvim config directory exists'
mkdir ~/.config/nvim/plugged || echo 'nvim plugged directory exists'

# Install plugged https://github.com/junegunn/vim-plug
# TODO I'm not sure if this is required or does autoload fix it?
# curl -fLo ~/.config/nvim/plugged/plug.vim --create-dirs \
#    https://raw.githubusercontent.com/junegunn/vim-plug/master/plug.vim
ln -s "$(pwd)/nvim/autoload" ~/.config/nvim/autoload
ln -s "$(pwd)/nvim/init.vim" ~/.config/nvim/init.vim

# Plugins. Run twice so themes are all set up.
nvim --headless +PlugInstall +qa
nvim --headless +PlugInstall +qa

