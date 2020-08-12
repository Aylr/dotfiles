
show_help() {
    case $1 in
        -h|--help)
        cat << DOCSTRING
Journaler.

If no date is specified, today will be used. Input can be piped in or added as
a positional argument in a string. Piped input takes precedence.

Arguments:
-d --date         desired date for entry
-b --base_dir     base directory to create files

Options
-e --edit         open entry in editor - uses \$EDITOR if available
-h --help         show this help and exit
-v --verbose      show verbose output

Usage

journal.sh                              View today's entry
journal.sh -e                           Edit today's entry
journal.sh "TIL something neat"         Add a note to today's entry
journal.sh "TIL something neat" -e      Add a note to today's entry and edit
journal.sh -d 2020-01-01                View the entry from 1/1/20
cat file.txt | journal.sh               Pipe file.txt into today's entry
DOCSTRING
        exit
        ;;
    esac
}

# Bash argument parsing https://stackoverflow.com/questions/192249/how-do-i-parse-command-line-arguments-in-bash
POSITIONAL=()
while [[ $# -gt 0 ]]
do
    key="$1"
    show_help $key

    case $key in
        -d|--date)
        TARGET_DATE="$2"
        shift # past argument
        shift # past value
        ;;
        -b|--base_dir)
        BASE_DIR="$2"
        shift # past argument
        shift # past value
        ;;
        -e|--edit)
        OPEN_EDITOR=YES
        shift # past argument
        ;;
        -v|--verbose)
        VERBOSE=YES
        shift # past argument
        ;;
        *)    # unknown option
        POSITIONAL+=("$1") # save it in an array for later
        shift # past argument
        ;;
    esac
done
set -- "${POSITIONAL[@]}" # restore positional parameters

BASE_DIR=${BASE_DIR:-"$HOME/repos/journal"}
TARGET_DATE=${TARGET_DATE:-$(date "+%Y-%m-%d")}
FILENAME="$BASE_DIR/$TARGET_DATE.md"
HEADER="# $TARGET_DATE\n\n"

# Input handling
if [ -p /dev/stdin ]; then
    # input is piped in
    NOTE=$(cat)
elif [ "$1" != "" ]; then 
    # input given as argument
    NOTE=$1
else
    NOTE=""
fi

if [ -n "$VERBOSE" ]; then
    echo "BASE_DIR          ${BASE_DIR}"
    echo "TARGET_DATE       ${TARGET_DATE}"
    echo "FILENAME          ${FILENAME}"
    echo "HEADER            ${HEADER}"
    echo "OPEN_EDITOR       ${OPEN_EDITOR}"
    echo "NOTE              ${NOTE}"
    echo "positional        ${POSITIONAL[*]}"
fi

# File handling
if [ -f $FILENAME ]; then
    # file exists
    if !( grep $TARGET_DATE $FILENAME -q ); then
        # TARGET_DATE header not in file --> insert it at beginning of file
        echo "$HEADER$(cat $FILENAME)" > $FILENAME
    fi
else
    # file doesn't exist --> create it
    echo $HEADER > $FILENAME
fi

if [ -n "$NOTE" ]; then echo "$NOTE" >> $FILENAME; fi
if [ -n "$OPEN_EDITOR" ]; then
	if [ -n "$EDITOR" ]; then
		$EDITOR $FILENAME;
	else
		nvim $FILENAME;
	fi
else
	cat $FILENAME;
fi
