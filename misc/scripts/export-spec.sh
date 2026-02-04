#!/bin/sh

BASE_URI=http://localhost:8080
OUT_DIR=../../ragbag-api
OUT_FILE="$OUT_DIR/openapi.yaml"

mkdir -p $OUT_DIR

echo "Exporting API specification from $BASE_URI/api/docs.yaml..."
curl "$BASE_URI/api/docs.yaml" -o $OUT_FILE
echo "API specification exported to $OUT_FILE"
