#!/bin/bash
# Creates the products index with kuromoji + ICU analyzers

ES_URL="${ES_URL:-http://localhost:9200}"
INDEX_NAME="${INDEX_NAME:-rakutao_products}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
MAPPING_FILE="$SCRIPT_DIR/../internal/search/index.json"

echo "Creating index $INDEX_NAME on $ES_URL..."

curl -s -X PUT "$ES_URL/$INDEX_NAME" \
  -H "Content-Type: application/json" \
  -d @"$MAPPING_FILE"

echo ""
echo "Index $INDEX_NAME created."
