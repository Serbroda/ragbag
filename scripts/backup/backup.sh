#!/bin/bash

POSITIONAL=()
while [[ $# -gt 0 ]]
do
key="$1"

case $key in
    -i|--input)
    INPUT_DIR="$2"
    shift # past argument
    shift # past value
    ;;
    -o|--output)
    OUT_DIR="$2"
    shift # past argument
    shift # past value
    ;;
    *)    # unknown option
    POSITIONAL+=("$1") # save it in an array for later
    shift # past argument
    ;;
esac
done
set -- "${POSITIONAL[@]}" # restore positional parameters

NOW=$(date +'%Y-%m-%d_%H%M%S')
FILENAME=backup_$NOW.tar.gz

echo Backup
echo ---------------

if [ -z "$INPUT_DIR" ]
then
    read -p 'Source Directory: ' INPUT_DIR
fi

if [ -z "$OUT_DIR" ]
then
    read -p 'Ouput Directory: ' OUT_DIR
fi

echo $INPUT_DIR
echo $OUT_DIR
echo $FILENAME

mkdir -p $OUT_DIR

tar -cvpzf "$OUT_DIR/$FILENAME" -C $INPUT_DIR .