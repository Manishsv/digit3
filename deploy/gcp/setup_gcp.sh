#!/bin/bash

# ======= CONFIGURE THIS ========
PROJECT_ID="digit-platform-456501"
REGION="us-central1"
SERVICE_ACCOUNT_NAME="github-deployer"
ARTIFACT_REGISTRY_NAME="digit-docker-repo"
CLOUD_SQL_INSTANCE_NAME="account-db"
# ===============================

echo "🔧 Setting project"
gcloud config set project "$PROJECT_ID"

echo "🔧 Enabling required services..."
gcloud services enable \
  run.googleapis.com \
  sqladmin.googleapis.com \
  artifactregistry.googleapis.com \
  iam.googleapis.com

echo "👤 Creating service account: $SERVICE_ACCOUNT_NAME"
gcloud iam service-accounts create "$SERVICE_ACCOUNT_NAME" \
  --description="GitHub Actions deployer" \
  --display-name="GitHub Deployer"

SA_EMAIL="$SERVICE_ACCOUNT_NAME@$PROJECT_ID.iam.gserviceaccount.com"

echo "🔐 Assigning roles to service account"
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$SA_EMAIL" \
  --role="roles/run.admin"

gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$SA_EMAIL" \
  --role="roles/storage.admin"

gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$SA_EMAIL" \
  --role="roles/iam.serviceAccountUser"

gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$SA_EMAIL" \
  --role="roles/cloudsql.client"

echo "📦 Creating Artifact Registry (Docker)"
gcloud artifacts repositories create "$ARTIFACT_REGISTRY_NAME" \
  --repository-format=docker \
  --location="$REGION" \
  --description="Docker repo for DIGIT services"

echo "🗝️  Creating service account key"
gcloud iam service-accounts keys create "./gcp-key-$PROJECT_ID.json" \
  --iam-account="$SA_EMAIL"

echo "✅ Setup complete!"
echo "➡️  Next steps:"
echo "1. Create Cloud SQL instance via console or CLI"
echo "2. Encode and upload gcp-key-$PROJECT_ID.json to GitHub as GCP_SA_KEY secret"
echo "3. Deploy using GitHub Actions 🚀"