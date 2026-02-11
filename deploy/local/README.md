# DIGIT 3.0 - Docker Compose Setup

This repository allows you to run the complete **DIGIT 3.0 core stack** locally using Docker Compose. It includes all essential services like MDMS, account, Workflow,idgen, Filestore, and more — all preconfigured for local development and exploration.

---

## Prerequisites

Ensure the following are installed on your machine:

- **Docker**: [Install Docker](https://docs.docker.com/get-docker/)
- **Docker Compose**: Comes with Docker Desktop.

## Clone the Repo

```bash
git clone https://github.com/digitnxt/digit3.git
cd digit3
git checkout develop
cd deploy/local

## Spin up Digit 3.0

docker compose up -d

## Cleanup

docker compose down -v

delete the volumes if you face migration issues. It might fail if there is network issue, do a cleanup using below command in that case too and try "docker compose up -d" again-
docker system prune -f --volumes (⚠️ be cautious — this removes all unused containers, images, volumes)

## Update etc/hosts(Only if you need rbac and running via kong gateway. for testing and development use the core services directly to avoid complications)

Run in terminal: echo "127.0.0.1 keycloak" | sudo tee -a /etc/hosts(for windows update it manually in C:\Windows\System32\drivers\etc\hosts
)

