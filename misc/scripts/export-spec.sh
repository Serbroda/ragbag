#!/bin/sh

BASE_URI=http://localhost:8080
OUT_DIR=../../ragbag-api
OUT_FILE="$OUT_DIR/openapi.json"

mkdir -p $OUT_DIR

curl "$BASE_URI/api/docs" -o $OUT_FILE
echo "API specification exported to $OUT_FILE"
