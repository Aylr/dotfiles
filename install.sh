#!/bin/bash

# install oh-my-zsh and preferred theme
echo "Installing oh-my-zsh"
sh -c "$(curl -fsSL https://raw.github.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"
echo "Installing Powerlevel10k"
git clone --depth=1 https://github.com/romkatv/powerlevel10k.git ${ZSH_CUSTOM:-$HOME/.oh-my-zsh/custom}/themes/powerlevel10k
# Powwerlevel 10k Fonts
echo "Installing Powerlevel10k Fonts..."
cd ~/Library/Fonts && {
    curl -O https://raw.githubusercontent.com/romkatv/powerlevel10k-media/master/MesloLGS%20NF%20Regular.ttf
    curl -O https://raw.githubusercontent.com/romkatv/powerlevel10k-media/master/MesloLGS%20NF%20Bold.ttf
    curl -O https://raw.githubusercontent.com/romkatv/powerlevel10k-media/master/MesloLGS%20NF%20Italic.ttf
    curl -O https://raw.githubusercontent.com/romkatv/powerlevel10k-media/master/MesloLGS%20NF%20Bold%20Italic.ttf
cd -; }

# Install brew if it is not installed
echo "Installing brew"
brew || /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Set up symlinks
echo "Setting up shell profile symlinks"
mv ~/.zshrc ~/.zshrc_backup_"$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
ln -s "$(pwd)"/zsh/.zshrc ~/.zshrc
ln -s "$(pwd)"/zsh/.p10k.zsh ~/.p10k.zsh
ln -s "$(pwd)"/bash/.bash_profile ~/.bash_profile
ln -s "$(pwd)"/bash/.gitconfig ~/.gitconfig
