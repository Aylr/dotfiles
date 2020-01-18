source /Users/taylor/repos/dotfiles/bash/.prompt
source /Users/taylor/repos/dotfiles/bash/.alias
source /Users/taylor/repos/dotfiles/bash/.dbt-completion.bash
source /Users/taylor/repos/dotfiles/bash/.autocompletion
source /Users/taylor/repos/dotfiles/bash/.infinite_history
source /Users/taylor/repos/dotfiles/bash/.exports
source /Users/taylor/repos/dotfiles/bash/.secrets


# added by Anaconda3 5.3.1 installer
# >>> conda init >>>
# !! Contents within this block are managed by 'conda init' !!
__conda_setup="$(CONDA_REPORT_ERRORS=false '/Users/taylor/anaconda3/bin/conda' shell.bash hook 2> /dev/null)"
if [ $? -eq 0 ]; then
    \eval "$__conda_setup"
else
    if [ -f "/Users/taylor/anaconda3/etc/profile.d/conda.sh" ]; then
        . "/Users/taylor/anaconda3/etc/profile.d/conda.sh"
        CONDA_CHANGEPS1=false conda activate base
    else
        \export PATH="/Users/taylor/anaconda3/bin:$PATH"
    fi
fi
unset __conda_setup
# <<< conda init <<<
