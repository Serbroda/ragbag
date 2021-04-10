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
    read -p 'Source Directory: ' INPUT_FILE
fi

if [ -z "$OUT_DIR" ]
then
    read -p 'Ouput Directory: ' OUT_DIR
fi

echo $INPUT_FILE
echo $OUT_DIR
echo $FILENAME

mkdir -p $OUT_DIR

tar -xzvf $INPUT_FILE -C $OUT_DIR