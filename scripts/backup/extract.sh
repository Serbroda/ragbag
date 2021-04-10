#!/bin/bash

POSITIONAL=()
while [[ $# -gt 0 ]]
do
key="$1"

case $key in
    -i|--input)
    INPUT_FILE="$2"
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

echo Extract backup
echo ---------------

if [ -z "$INPUT_FILE" ]
then
    read -p 'Backup file to extract: ' INPUT_FILE
else
    echo "Backup file to extract: : $INPUT_FILE"
fi

if [ -z "$OUT_DIR" ]
then
    read -p 'Ouput directory: ' OUT_DIR
else
    echo "Ouput directory: : $OUT_DIR"
fi

echo
echo "Starting extraction $INPUT_FILE -> $OUT_DIR ..."

mkdir -p $OUT_DIR

tar -xzvf $INPUT_FILE -C $OUT_DIR

echo 
echo Extraction successfull
