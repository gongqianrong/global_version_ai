#!/bin/bash
set -e

# Creates the products index with kuromoji + ICU analyzers

ES_URL="${ES_URL:-http://localhost:9200}"
INDEX_NAME="${INDEX_NAME:-rakutao_products}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
MAPPING_FILE="$SCRIPT_DIR/../internal/search/index.json"

# Wait for Elasticsearch to be ready.
echo "Waiting for Elasticsearch at $ES_URL..."
MAX_RETRIES=30
RETRY_COUNT=0
until curl -s "$ES_URL" > /dev/null 2>&1; do
  RETRY_COUNT=$((RETRY_COUNT + 1))
  if [ "$RETRY_COUNT" -ge "$MAX_RETRIES" ]; then
    echo "ERROR: Elasticsearch not available after $MAX_RETRIES attempts."
    exit 1
  fi
  echo "  Elasticsearch not ready, retrying in 2s... ($RETRY_COUNT/$MAX_RETRIES)"
  sleep 2
done
echo "Elasticsearch is ready."

# Check if the index already exists.
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$ES_URL/$INDEX_NAME")
if [ "$HTTP_CODE" = "200" ]; then
  echo "Index $INDEX_NAME already exists, skipping creation."
  exit 0
fi

echo "Creating index $INDEX_NAME on $ES_URL..."

RESPONSE=$(curl -s -w "\n%{http_code}" -X PUT "$ES_URL/$INDEX_NAME" \
  -H "Content-Type: application/json" \
  -d @"$MAPPING_FILE")

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" -ge 200 ] && [ "$HTTP_CODE" -lt 300 ]; then
  echo "Index $INDEX_NAME created successfully."
else
  echo "ERROR: Failed to create index $INDEX_NAME (HTTP $HTTP_CODE)."
  echo "$BODY"
  exit 1
fi
