#!/bin/bash
set -e

# Database initialization script for image-service-go
# This script creates the database, user, and applies migrations

# Default values
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-image_service}"
DB_USER="${DB_USER:-image_user}"
DB_PASSWORD="${DB_PASSWORD:-image_password}"
MIGRATIONS_DIR="$(dirname "$(dirname "$0")")/migrations"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Print usage information
usage() {
  echo "Usage: $0 [options]"
  echo ""
  echo "Options:"
  echo "  --host HOST         Database host (default: $DB_HOST)"
  echo "  --port PORT         Database port (default: $DB_PORT)"
  echo "  --name NAME         Database name (default: $DB_NAME)"
  echo "  --user USER         Database user (default: $DB_USER)"
  echo "  --password PASSWORD Database password (default: $DB_PASSWORD)"
  echo "  --migrations DIR    Migrations directory (default: $MIGRATIONS_DIR)"
  echo "  --help              Show this help message"
  echo ""
  echo "Environment variables:"
  echo "  DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD can be used instead of command line options"
  exit 1
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --host)
      DB_HOST="$2"
      shift 2
      ;;
    --port)
      DB_PORT="$2"
      shift 2
      ;;
    --name)
      DB_NAME="$2"
      shift 2
      ;;
    --user)
      DB_USER="$2"
      shift 2
      ;;
    --password)
      DB_PASSWORD="$2"
      shift 2
      ;;
    --migrations)
      MIGRATIONS_DIR="$2"
      shift 2
      ;;
    --help)
      usage
      ;;
    *)
      echo "Unknown option: $1"
      usage
      ;;
  esac
done

# Check for required tools
check_command() {
  if ! command -v "$1" &> /dev/null; then
    echo -e "${RED}Error: $1 is required but not installed.${NC}"
    exit 1
  fi
}

check_command psql

# Function to run SQL commands as postgres user
run_psql() {
  PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -p "$DB_PORT" -U postgres -c "$1"
}

# Function to run SQL commands on the specific database
run_psql_db() {
  PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -p "$DB_PORT" -U postgres -d "$DB_NAME" -c "$1"
}

echo -e "${YELLOW}Initializing database for image-service-go...${NC}"

# Check if we can connect to PostgreSQL
echo -e "Checking PostgreSQL connection..."
if ! PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -p "$DB_PORT" -U postgres -c '\q' 2>/dev/null; then
  echo -e "${RED}Error: Could not connect to PostgreSQL server.${NC}"
  echo "Please make sure PostgreSQL is running and the postgres user is available."
  exit 1
fi

# Create database if it doesn't exist
echo -e "Creating database $DB_NAME if it doesn't exist..."
if ! run_psql "SELECT 1 FROM pg_database WHERE datname = '$DB_NAME'" | grep -q 1; then
  run_psql "CREATE DATABASE $DB_NAME;"
  echo -e "${GREEN}Database $DB_NAME created.${NC}"
else
  echo -e "Database $DB_NAME already exists."
fi

# Create user if it doesn't exist
echo -e "Creating user $DB_USER if it doesn't exist..."
if ! run_psql "SELECT 1 FROM pg_roles WHERE rolname = '$DB_USER'" | grep -q 1; then
  run_psql "CREATE USER $DB_USER WITH ENCRYPTED PASSWORD '$DB_PASSWORD';"
  echo -e "${GREEN}User $DB_USER created.${NC}"
else
  echo -e "User $DB_USER already exists."
  # Update password in case it changed
  run_psql "ALTER USER $DB_USER WITH ENCRYPTED PASSWORD '$DB_PASSWORD';"
fi

# Grant privileges to the user
echo -e "Granting privileges to $DB_USER on $DB_NAME..."
run_psql "GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;"
run_psql_db "GRANT ALL PRIVILEGES ON SCHEMA public TO $DB_USER;"
echo -e "${GREEN}Privileges granted.${NC}"

# Apply migrations
echo -e "Applying migrations from $MIGRATIONS_DIR..."

# Check if migrate tool is available
if command -v migrate &> /dev/null; then
  # Using golang-migrate
  migrate -path "$MIGRATIONS_DIR" -database "postgres://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=disable" up
  echo -e "${GREEN}Migrations applied using migrate tool.${NC}"
else
  # Fallback to manual migration if migrate tool is not available
  echo -e "${YELLOW}Warning: 'migrate' tool not found, applying migrations manually.${NC}"
  
  # Create migrations table if it doesn't exist
  run_psql_db "CREATE TABLE IF NOT EXISTS schema_migrations (version bigint NOT NULL, dirty boolean NOT NULL, PRIMARY KEY (version));"
  
  # Apply each migration file
  for migration_file in "$MIGRATIONS_DIR"/*.sql; do
    if [ -f "$migration_file" ]; then
      echo -e "Applying migration: $(basename "$migration_file")..."
      # Extract migration version from filename
      version=$(basename "$migration_file" | cut -d'_' -f1)
      
      # Check if migration already applied
      if ! run_psql_db "SELECT 1 FROM schema_migrations WHERE version = $version" | grep -q 1; then
        # Apply migration
        PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -p "$DB_PORT" -U postgres -d "$DB_NAME" -f "$migration_file"
        # Record migration
        run_psql_db "INSERT INTO schema_migrations (version, dirty) VALUES ($version, false);"
        echo -e "${GREEN}Migration $version applied.${NC}"
      else
        echo -e "Migration $version already applied."
      fi
    fi
  done
fi

echo -e "${GREEN}Database initialization complete!${NC}"
echo -e "You can now connect to the database using:"
echo -e "  psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME"

exit 0
