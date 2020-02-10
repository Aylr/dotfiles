JOURNAL_DIR="$HOME/repos/journal"
NOTE=""

# Parse args
while [ $# -gt 0 ]; do
    if [[ $1 == *"--"* ]]; then
        param="${1/--/}"
        if [ $param == "date" ]; then
            date_stamp="$2"
        else
            declare $param="$2"
        fi
        # echo $param $2
    fi
    shift
done

date_stamp=${date_stamp:-$(date "+%Y-%m-%d")}
FILENAME="$JOURNAL_DIR/$date_stamp.md"
HEADER="# $date_stamp\n\n- "

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
    if !( grep $date_stamp $FILENAME -q ); then
        # date_stamp header not in file --> insert it at beginning of file
        echo "$HEADER$(cat $FILENAME)" > $FILENAME
    fi
else
    # file doesn't exist --> create it
    echo $HEADER > $FILENAME
fi

if [ -n "$NOTE" ]; then echo $NOTE >> $FILENAME; fi
cat $FILENAME
