JOURNAL_DIR="$HOME/repos/journal"
DATE=$(date "+%Y-%m-%d")
FILENAME="$JOURNAL_DIR/$DATE.md"
HEADER="# $DATE\n\n- "
NOTE=""

# Input handling
if [ -p /dev/stdin ]; then
    # input is piped in
    NOTE=$(cat)
elif [ "$1" != "" ]; then 
    # input given as argument
    NOTE=$1
fi

# File handling
if [ -f $FILENAME ]; then
    # file exists
    if !( grep $DATE $FILENAME -q ); then
        # date header not in file --> insert it at beginning of file
        echo "$HEADER$(cat $FILENAME)" > $FILENAME
    fi
else
    # file doesn't exist --> create it
    echo $HEADER > $FILENAME
fi
echo $NOTE >> $FILENAME
cat $FILENAME
