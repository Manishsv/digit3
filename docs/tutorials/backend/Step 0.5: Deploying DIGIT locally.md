# DIGIT Deployment Guide

## Overview

DIGIT is a modular, multi-tenant digital public infrastructure platform, consisting of several frontend applications and backend microservices. It can be deployed across local developer machines, shared sandbox environments, and production environments (cloud or on-prem).

---

## Hardware Requirements

DIGIT includes multiple microservices (Account, Identity, Registry, Registration, Workflow, etc.) and Flutter-based frontend applications. Running the full stack locally is resource-intensive. You'll need:

- **Minimum**: 16 GB RAM, 4-core CPU
- **Recommended**: 32 GB RAM, 8-core CPU with SSD storage

---

## Local Development using Docker Compose

Follow along with [this video](https://github.com/digitnxt/digit3/blob/65df8ded086bead4ca09386c5b89172835506417/Deploy%20DIGIT%20w%20Docker%20(5).mp4)

DIGIT provides a docker-compose setup for local development.

### Prerequisites
- Docker Engine ≥ 24.x
- Docker Compose ≥ 2.x
- Make (optional, for convenience)
- Git

### Setup Instructions

```bash
# Clone the DIGIT repository
git clone https://github.com/egovernments/DIGIT.git
cd DIGIT

# Start all services using docker-compose
docker-compose up -d

# To tail logs
docker-compose logs -f

# To shut down services
docker-compose down
```

### Access Applications
- **Console UI**: http://localhost:3000
- **Admin UI**: http://localhost:3001
- **APIs** (e.g., Account): http://localhost:8080/api/account/v1/

---

## Shared Sandbox Environment

A shared DIGIT Sandbox is available for developers who don't want to run the full stack locally.

### Benefits
- Pre-configured services
- Shared infrastructure
- Secure access for testing and integration
- Extend existing services or build new ones

### How to Use Sandbox
1. Visit: https://sandbox.digit.org
2. Sign up and create your account
3. Use provided APIs and admin panels to register and manage services
4. Secure APIs via sandbox-issued credentials

> **Note**: All usage is monitored and rate-limited for fair access.

---

## Development & Production Deployment

DIGIT can be deployed in both public cloud and private data center environments.

### Cloud Deployment (AWS, GCP, Azure)

DIGIT includes deployment scripts and Terraform modules for major cloud providers.

#### AWS Example

```bash
cd deploy/aws
terraform init
terraform apply
```

- Creates VPC, ECS, RDS (Postgres), S3, etc.
- Deploys all DIGIT services as containers on ECS
- Configurable auto-scaling, logging, and monitoring

#### GCP Example

```bash
cd deploy/gcp
terraform init
terraform apply
```

- Uses GKE for container orchestration
- Google Cloud SQL for PostgreSQL
- IAM roles for secure service-to-service communication

#### Azure Example

```bash
cd deploy/azure
terraform init
terraform apply
```

- Uses Azure Kubernetes Service (AKS)
- Azure Blob for file service
- Integrated with Azure AD for Identity

---

## Private Data Center Deployment (On-Prem)

DIGIT supports on-prem deployments using Docker Swarm.

### Setup Using Docker Swarm

```bash
# Initialize swarm
docker swarm init

# Deploy DIGIT stack
docker stack deploy -c docker-compose.yml digit

# Check services
docker service ls

# Scale or update services
docker service scale digit_account=3
```

- Includes service discovery and load balancing
- Supports persistent storage and secure overlay networks

---

## Customization

You can override environment variables, secrets, and volumes via .env and override files.

```bash
cp env.example .env
vi .env
```

You can also create new services by:
1. Forking a template service
2. Registering it with the Catalogue service
3. Creating workflows and schemas using DIGIT Studio or APIs
