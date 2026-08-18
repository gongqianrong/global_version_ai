#!/bin/sh
set -e

echo "🚀 Starting Rakutao Gateway..."

# Wait for PostgreSQL to be ready
echo "⏳ Waiting for PostgreSQL..."
until psql "$DATABASE_URL" -c '\q' 2>/dev/null; do
  echo "PostgreSQL is unavailable - sleeping"
  sleep 2
done
echo "✅ PostgreSQL is ready"

# Run database migrations
echo "📦 Running database migrations..."
if [ -d "/app/migrations" ]; then
  for migration in /app/migrations/*.sql; do
    if [ -f "$migration" ]; then
      echo "   Executing: $(basename $migration)"
      psql "$DATABASE_URL" -f "$migration" 2>&1 | grep -v "already exists" | grep -v "NOTICE" || true
    fi
  done
  echo "✅ Migrations completed"
else
  echo "⚠️  No migrations directory found, skipping..."
fi

echo "🎯 Starting gateway service..."
echo ""

# Execute the main command
exec "$@"
