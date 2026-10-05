<div align="center">

# WaveFlix Hub

![Go](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&logoColor=white) ![Node.js](https://img.shields.io/badge/Node.js-20%2B-339933?logo=nodedotjs&logoColor=white) ![Next.js](https://img.shields.io/badge/Next.js-16-000000?logo=next.js&logoColor=white) ![License](https://img.shields.io/badge/License-MIT-blue.svg)

A self-hosted video streaming monorepo with a Go backend, Next.js clients, content extraction, and HLS media services.

[Features](#features) · [Requirements](#requirements) · [Run locally](#run-locally) · [Project layout](#project-layout) · [License](#license)

</div>

---

## Features

- Go API with SQLite and PostgreSQL support.
- Next.js streaming application and project website.
- Extractor and HLS services for media processing and delivery.
- npm workspaces and Turborepo for JavaScript projects.
- Docker Compose and Kubernetes configuration for deployment.
- Operational guides for backups and disaster recovery.

## Requirements

- Node.js 20 or newer and npm 11 or newer
- Go 1.25 or newer
- Docker Compose for container-based development

## Run locally

Clone the repository and install the JavaScript dependencies:

```bash
git clone https://github.com/achmad-miftahurrojak/waveflix-hub.git
cd waveflix-hub
npm install
```

Copy the relevant environment template before starting a service:

```bash
cp waveflix-app/backend/.env.example waveflix-app/backend/.env
```

Start the JavaScript workspaces:

```bash
npm run dev
```

Run the backend separately when needed:

```bash
cd waveflix-app/backend
go run .
```

Build the JavaScript workspaces:

```bash
npm run build
```

For the container setup, configure the variables used by `docker-compose.yml`, then run:

```bash
docker compose up --build
```

## Configuration and security

Use the checked-in `.env.example` files as templates. Keep populated `.env` files, API keys, passwords, private keys, TLS certificates, Kubernetes Secret manifests, local databases, logs, and generated binaries out of Git.

Production credentials belong in the deployment platform's secret manager. Replace every `REPLACE_WITH_*` value before deployment, and provide `JWT_SECRET`, database credentials, Redis credentials, and third-party API keys through the environment.

## Project layout

```text
waveflix-app/
├── backend/        # Go API, authentication, and database adapters
└── frontend/       # Next.js streaming application
waveflix-web/       # Next.js project website
waveflix-extractor/ # Content extraction service
waveflix-cdn/       # HLS and media processing services
database/           # Database setup and migration files
docs/               # Operational and recovery guides
k8s/                # Kubernetes manifests
docker-compose.yml  # Local container setup
```

## License

[MIT](LICENSE)
