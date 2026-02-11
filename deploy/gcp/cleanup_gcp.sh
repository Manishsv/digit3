#!/bin/bash

# ---- CONFIGURE THESE ----
PROJECT_ID="digit-platform-456501"
REGION="us-central1"
SERVICE_ACCOUNT_NAME="github-deployer"
ARTIFACT_REGISTRY="digit-docker-repo"
KEY_FILE="gcp-key-$PROJECT_ID.json"
# -------------------------

SA_EMAIL="$SERVICE_ACCOUNT_NAME@$PROJECT_ID.iam.gserviceaccount.com"

echo "🧹 Cleaning up GCP resources for project: $PROJECT_ID"

# Confirm project
gcloud config set project "$PROJECT_ID"

# Delete service account key (if file exists)
if [[ -f "$KEY_FILE" ]]; then
  echo "🔸 Deleting local key file: $KEY_FILE"
  rm "$KEY_FILE"
else
  echo "ℹ️  Key file not found locally: $KEY_FILE"
fi

# List keys and delete them remotely
echo "🔸 Deleting service account keys from GCP..."
KEY_IDS=$(gcloud iam service-accounts keys list \
  --iam-account="$SA_EMAIL" \
  --format="value(name)")

for KEY_ID in $KEY_IDS; do
  echo "   🔻 Deleting key: $KEY_ID"
  gcloud iam service-accounts keys delete "$KEY_ID" \
    --iam-account="$SA_EMAIL" --quiet
done

# Delete the service account
echo "🔸 Deleting service account: $SA_EMAIL"
gcloud iam service-accounts delete "$SA_EMAIL" --quiet || echo "⚠️  Could not delete service account"

# Delete Artifact Registry
read -p "❓ Delete Artifact Registry [$ARTIFACT_REGISTRY] in $REGION? (y/n): " CONFIRM
if [[ "$CONFIRM" == "y" || "$CONFIRM" == "Y" ]]; then
  echo "🔸 Deleting Artifact Registry: $ARTIFACT_REGISTRY"
  gcloud artifacts repositories delete "$ARTIFACT_REGISTRY" \
    --location="$REGION" --quiet
else
  echo "⏩ Skipping deletion of Artifact Registry"
fi

# (Optional) Disable APIs — commented out for safety
# echo "🔸 Disabling enabled services..."
# gcloud services disable cloudbuild.googleapis.com run.googleapis.com artifactregistry.googleapis.com iam.googleapis.com

echo "✅ Cleanup complete!"