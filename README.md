# dotfiles for command line superpowers

Organized dotfiles give you command line superpowers.

This was heavily inspired by [Getting Started With Dotfiles](https://medium.com/@webprolific/getting-started-with-dotfiles-43c3602fd789), which is a must-read. 

## Installation

1. `git clone git@github.com:Aylr/dotfiles.git`
2. **IMPORTANT** Backup your existing `~/.bash_profile` and `~/.gitconfig` or you **will have data loss.**
3. In the repo's `.bash_profile`, update the full path to this repo, including the trailing slash.
    - `DOTFILE_DIR=/Users/taylor/repos/dotfiles/`
3. From the repo directory, run `install.sh`. All this really does is drop symlinks into your home directory, which you can easily do yourself **after you backup existing files!**

## Notes

- **Non secret** exports are kept in `bash/.exports`
- **Secret exports** are kept in `bash/.secrets`, which are of course in the `.gitignore`

## References

- [Maximize Developer productivity on a Mac](https://medium.com/@ankushagarwal/maximize-developer-productivity-on-a-mac-a9ae6fbaedab)
