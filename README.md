# WaveFlix Hub

A self-hosted video streaming project with a Go backend, web clients, content extractors, and an HLS service.

[Features](#features) · [Architecture](#architecture) · [Build and run](#build-and-run) · [Project layout](#project-layout)

---

## Overview

WaveFlix Hub is a monorepo for developing and running the WaveFlix streaming platform. It includes a Go API, Next.js web applications, a content extractor, and supporting services for local or container-based deployment.

## Features

- Go backend with SQLite and PostgreSQL support.
- Next.js clients for the streaming app and project website.
- Extractor and HLS services for media processing and delivery.
- Docker Compose configurations for local and production-oriented setups.
- Kubernetes manifests and operational documentation.

## Architecture

```text
Browser ──> Next.js frontend ──> Go API ──> SQLite or PostgreSQL
                                   │
                                   ├──> Extractor service
                                   └──> HLS service ──> Media storage
```

The root `package.json` uses npm workspaces and Turborepo to coordinate the JavaScript services. Go services are built from their own modules.

## Build and run

### Requirements

- Node.js 20 or newer and npm
- Go 1.25 or newer
- Docker Compose for the container-based setup

### Install dependencies

```sh
git clone https://github.com/achmad-miftahurrojak/waveflix-hub.git
cd waveflix-hub
npm install
```

Copy the relevant `.env.example` file for the service you want to run, then set local database and API credentials. Keep populated `.env` files out of Git.

### Run services

Start the services configured for development:

```sh
npm run dev
```

Build the JavaScript workspaces:

```sh
npm run build
```

To start the container setup, configure the variables referenced by `docker-compose.yml` and run:

```sh
docker compose up --build
```

## Project layout

```text
waveflix-hub/
├── waveflix-app/
│   ├── backend/       # Go API and database adapters
│   └── frontend/      # Next.js streaming application
├── waveflix-web/      # Next.js project website
├── waveflix-extractor/ # Content extraction service
├── waveflix-cdn/      # HLS media service
├── database/          # Database setup and migration files
├── docker-compose.yml # Local container setup
├── k8s/               # Kubernetes deployment manifests
└── docs/              # Operations and recovery guides
```

## Configuration and security

Use the checked-in `.env.example` files as templates and provide credentials through local environment files or your deployment's secret manager. Kubernetes Secret manifests containing real values, private keys, and generated binaries are not source files and should stay out of Git.

## License

See [LICENSE](LICENSE).
