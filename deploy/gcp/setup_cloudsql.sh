#!/bin/bash

# --- CONFIGURATION ---
# You need a .env file in the same directory containing:
#   DB_PASSWORD=yourpassword
#
# Optionally, you can add additional variables to the .env file
#
# For example, your .env file might contain:
#   DB_PASSWORD=superSecretPass123
#   PROJECT_ID=your-project-id
#   REGION=us-central1
#   INSTANCE_NAME=account-db
#   DB_NAME=account
#   DB_USER=account_user
#
# If any of these are not set in the .env file, the script will use these defaults:
DEFAULT_PROJECT_ID="your-project-id"
DEFAULT_REGION="us-central1"
DEFAULT_INSTANCE_NAME="account-db"
DEFAULT_DB_NAME="account"
DEFAULT_DB_USER="account_user"
DEFAULT_DB_VERSION="POSTGRES_15"
DEFAULT_MACHINE_TYPE="db-f1-micro"
# ---------------------

# Check if .env file exists in the current directory
if [ ! -f .env ]; then
  echo "ERROR: .env file not found! Please create a .env file with at least DB_PASSWORD set."
  exit 1
fi

# Source the .env file
source .env

# Check that DB_PASSWORD is set
if [ -z "$DB_PASSWORD" ]; then
  echo "ERROR: DB_PASSWORD is not set in the .env file!"
  exit 1
fi

# Use values from .env if provided, or fallback to defaults
PROJECT_ID="${PROJECT_ID:-$DEFAULT_PROJECT_ID}"
REGION="${REGION:-$DEFAULT_REGION}"
INSTANCE_NAME="${INSTANCE_NAME:-$DEFAULT_INSTANCE_NAME}"
DB_NAME="${DB_NAME:-$DEFAULT_DB_NAME}"
DB_USER="${DB_USER:-$DEFAULT_DB_USER}"
DB_VERSION="${DB_VERSION:-$DEFAULT_DB_VERSION}"
MACHINE_TYPE="${MACHINE_TYPE:-$DEFAULT_MACHINE_TYPE}"

echo "🔧 Setting project to $PROJECT_ID"
gcloud config set project "$PROJECT_ID"

echo "🛠️  Creating Cloud SQL instance: $INSTANCE_NAME"
gcloud sql instances create "$INSTANCE_NAME" \
  --database-version="$DB_VERSION" \
  --tier="$MACHINE_TYPE" \
  --region="$REGION" \
  --root-password="$DB_PASSWORD"

echo "🗃️  Creating database: $DB_NAME"
gcloud sql databases create "$DB_NAME" \
  --instance="$INSTANCE_NAME"

echo "👤 Creating user: $DB_USER"
gcloud sql users create "$DB_USER" \
  --instance="$INSTANCE_NAME" \
  --password="$DB_PASSWORD"

echo "🔗 Fetching connection string"
CONNECTION_NAME=$(gcloud sql instances describe "$INSTANCE_NAME" --format="value(connectionName)")

echo
echo "✅ Cloud SQL PostgreSQL instance created!"
echo "➡️  Use this in your GitHub secret DATABASE_URL:"
echo
echo "postgres://$DB_USER:$DB_PASSWORD@localhost/$DB_NAME?host=/cloudsql/$CONNECTION_NAME"