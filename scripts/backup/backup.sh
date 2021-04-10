#!/bin/bash

echo Backup directory
---------------------------

FILENAME_PREFIX=backup

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
    -n|--name)
    FILENAME_PREFIX="$2"
    shift # past argument
    shift # past value
    ;;
    -y|--yes)
    yn="y"
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
FILENAME="$FILENAME_PREFIX"_"$NOW".tar.gz


if [ -z "$INPUT_DIR" ]
then
    read -e -p 'Source directory: ' INPUT_DIR
else
    echo "Source directory: $INPUT_DIR"
fi

if [ -z "$OUT_DIR" ]
then
    read -e -p 'Ouput directory: ' OUT_DIR
else
    echo "Ouput directory: $OUT_DIR"
fi

echo

if [ -z "$yn" ]
then
    while true; do
        echo Do you want to start the backup with following options?
        echo -----------------------------------------------------------
        echo "- Source directory to created backup for:" $INPUT_DIR
        echo "- Target backup file:" "$OUT_DIR/$FILENAME"
        echo 
        read -p "Start backup (Y/n): " yn

        if [ -z "$yn" ]
        then
            yn="y"
        fi

        case $yn in
            [Yy]* ) break;;
            [Nn]* ) exit;;
            * ) echo "Please answer yes or no.";;
        esac
    done
fi

echo
echo "Starting backup $INPUT_DIR -> $OUT_DIR/$FILENAME ..."
echo

mkdir -p $OUT_DIR

tar -cvpzf "$OUT_DIR/$FILENAME" -C $INPUT_DIR .

echo 
echo Backup sucessfully created
