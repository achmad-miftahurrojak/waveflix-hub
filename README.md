# WaveFlix Hub

A comprehensive Video-on-Demand platform featuring automated content extraction, scalable backend services, and responsive web interfaces.

![Go](https://img.shields.io/badge/Go-1.21-00ADD8?logo=go&logoColor=white)
![Node.js](https://img.shields.io/badge/Node.js-18%2B-339933?logo=nodedotjs&logoColor=white)
![React](https://img.shields.io/badge/React-18-61DAFB?logo=react&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-4169E1?logo=postgresql&logoColor=white)
![Turborepo](https://img.shields.io/badge/Turborepo-2-EF4444?logo=turborepo&logoColor=white)

## Table of Contents

1. [Features](#features)
2. [Screenshot](#screenshot)
3. [Getting Started](#getting-started)
4. [Usage](#usage)
5. [Directory Structure](#directory-structure)
6. [API Reference](#api-reference)
7. [Contributing](#contributing)
8. [License](#license)
9. [Contact](#contact)

## Features

- High-Performance Backend: API services written in Go for minimal latency.
- Automated Extractor: Programmatic content ingestion and processing tools.
- Modern Web Interface: React-based frontend for seamless user navigation.
- Monorepo Architecture: Coordinated multi-service development managed by Turborepo.
- Containerized Deployment: Docker and Kubernetes ready infrastructure.

## Screenshot

![WaveFlix Hub Demo](https://via.placeholder.com/800x450?text=WaveFlix+Hub+Demo)

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Node.js 18 or higher
- PostgreSQL database
- pnpm package manager

### Installation Steps

```bash
git clone https://github.com/hamin-baek/hamin-baek.git
cd software/waveflix-hub
pnpm install
```

### Configuration

Create a `.env.local` file in the relevant subdirectories (e.g., `waveflix-app/backend`, `waveflix-app/frontend`) to configure database credentials and external API keys.

## Usage

Start the entire ecosystem via Turborepo:
```bash
pnpm run dev
```

Build all services for production:
```bash
pnpm run build
```

## Directory Structure

- `waveflix-app/`: Core application logic.
  - `backend/`: Go-based RESTful API server.
  - `frontend/`: React-based user interface.
- `waveflix-extractor/`: Content scraping and ingestion scripts.
- `waveflix-web/`: Landing pages and administrative web panels.
- `k8s/`: Kubernetes deployment manifests.
- `docker-compose.yml`: Local orchestration configuration.

## API Reference

The Go backend exposes a REST API for content retrieval and user management. Ensure the server is running and access the `/api/docs` endpoint for the Swagger specification.

## Contributing

Follow the established branching model and ensure all Turborepo pipeline checks pass before submitting modifications.

## License

This project is licensed under the MIT License.

## Contact

Created by Achmad Miftahurrojak.
[GitHub](https://github.com/hamin-baek)
