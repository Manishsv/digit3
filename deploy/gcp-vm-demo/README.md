# DIGIT demo on GCP (single VM, minimal cost)

This runbook deploys:

- **DIGIT 3 core stack** via `deploy/local/docker-compose.yml`
- **DIGIT Provision Console** via `digit-client-tools/provision-console` (Node server that serves the UI and proxies API calls)

Target: **demo-only**, **no domain**, **no SMS/email**, **minimum exposed ports**.

## Quick start (create the VM)

On your laptop (must have `gcloud` authenticated):

```bash
export PROJECT_ID="digit-492716"
export REGION="us-central1"
export ZONE="us-central1-a"
export VM_NAME="digit-demo-1"

gcloud config set project "$PROJECT_ID"
gcloud services enable compute.googleapis.com

gcloud compute instances create "$VM_NAME" \
  --zone "$ZONE" \
  --machine-type "e2-standard-4" \
  --boot-disk-size "80GB" \
  --boot-disk-type "pd-standard" \
  --image-family "ubuntu-2404-lts" \
  --image-project "ubuntu-os-cloud" \
  --tags "digit-demo" \
  --metadata-from-file startup-script=./startup.sh

gcloud compute firewall-rules create digit-demo-allow-ssh \
  --direction=INGRESS --priority=1000 --network=default --action=ALLOW \
  --rules=tcp:22 --source-ranges=0.0.0.0/0 --target-tags=digit-demo || true

gcloud compute firewall-rules create digit-demo-allow-http \
  --direction=INGRESS --priority=1000 --network=default --action=ALLOW \
  --rules=tcp:80 --source-ranges=0.0.0.0/0 --target-tags=digit-demo || true

gcloud compute instances describe "$VM_NAME" --zone "$ZONE" \
  --format="value(networkInterfaces[0].accessConfigs[0].natIP)"
```

## Deploy DIGIT + Provision Console (on the VM)

SSH in:

```bash
gcloud compute ssh "$VM_NAME" --zone "$ZONE"
```

Clone repos:

```bash
mkdir -p ~/digit-demo && cd ~/digit-demo
git clone https://github.com/Manishsv/digit3 digit3
git clone https://github.com/Manishsv/digit-client-tools digit-client-tools
```

Bring up DIGIT core stack:

```bash
cd ~/digit-demo/digit3/deploy/local
docker compose up -d
```

Run full provision:

```bash
cd ~/digit-demo/digit3/deploy/local
./up-and-provision.sh
```

Run Provision Console (served via nginx on port 80):

```bash
cd ~/digit-demo/digit-client-tools/provision-console

docker run --rm -u "$(id -u):$(id -g)" \
  -v "$PWD:/app" -w /app node:20-bullseye \
  bash -lc "npm ci && npm run build"

docker run -d --restart unless-stopped \
  --name provision-console \
  --network host \
  -e NODE_ENV=production \
  -v "$PWD:/app" -w /app node:20-bullseye \
  bash -lc "npm ci --omit=dev && node server/index.mjs"
```

Open:

- `http://<VM_PUBLIC_IP>/`

## Teardown

```bash
gcloud compute instances delete "$VM_NAME" --zone "$ZONE"
gcloud compute firewall-rules delete digit-demo-allow-ssh digit-demo-allow-http
```

