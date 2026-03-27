# Enable Powerlevel10k instant prompt. Should stay close to the top of ~/.zshrc.
if [[ -r "${XDG_CACHE_HOME:-$HOME/.cache}/p10k-instant-prompt-${(%):-%n}.zsh" ]]; then
  source "${XDG_CACHE_HOME:-$HOME/.cache}/p10k-instant-prompt-${(%):-%n}.zsh"
fi

# ── Oh My Zsh ────────────────────────────────────────────────────────────────
export ZSH="$HOME/.oh-my-zsh"
ZSH_THEME=powerlevel10k/powerlevel10k
plugins=(z)
source $ZSH/oh-my-zsh.sh

# ── Dotfile modules ──────────────────────────────────────────────────────────
export DOTFILE_DIR="$HOME/repos/dotfiles/"
source ${DOTFILE_DIR}shell/.alias
source ${DOTFILE_DIR}shell/.exports
[ -f ${DOTFILE_DIR}shell/.secrets ] && source ${DOTFILE_DIR}shell/.secrets

# ── Prompt ───────────────────────────────────────────────────────────────────
[[ ! -f ~/.p10k.zsh ]] || source ~/.p10k.zsh

# ── Completions ──────────────────────────────────────────────────────────────
autoload -Uz compinit
zstyle ':completion:*' menu select
fpath+=~/.zfunc

# ── Tool integrations ────────────────────────────────────────────────────────
[ -f ~/.fzf.zsh ] && source ~/.fzf.zsh

# Google Cloud SDK (installed via brew cask or manual install)
if [ -f "$HOME/google-cloud-sdk/path.zsh.inc" ]; then
  . "$HOME/google-cloud-sdk/path.zsh.inc"
  [ -f "$HOME/google-cloud-sdk/completion.zsh.inc" ] && . "$HOME/google-cloud-sdk/completion.zsh.inc"
elif [ -f "$BREW_PREFIX/share/google-cloud-sdk/path.zsh.inc" ]; then
  . "$BREW_PREFIX/share/google-cloud-sdk/path.zsh.inc"
  [ -f "$BREW_PREFIX/share/google-cloud-sdk/completion.zsh.inc" ] && . "$BREW_PREFIX/share/google-cloud-sdk/completion.zsh.inc"
fi

[ -f "$HOME/.config/broot/launcher/bash/br" ] && source "$HOME/.config/broot/launcher/bash/br"

export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
[ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"

[ -f "$HOME/.local/bin/env" ] && . "$HOME/.local/bin/env"

# ── Extra PATH entries ───────────────────────────────────────────────────────
[ -d "$BREW_PREFIX/opt/openjdk/bin" ] && export PATH="$BREW_PREFIX/opt/openjdk/bin:$PATH"
[ -d "$HOME/.cargo/bin" ] && export PATH="$HOME/.cargo/bin:$PATH"
export PATH="$HOME/.local/bin:$PATH"
