# SecureMe

Endpoint security management platform — enroll Windows agents, push policies, collect events, and view device status from a web UI.

## Prerequisites

- [Docker 24+](https://docs.docker.com/get-docker/) with Compose v2
- [Go 1.22+](https://go.dev/dl/) (for local development)
- [Node 20+](https://nodejs.org/) (for local frontend development)

## Quick Start

```bash
git clone https://github.com/samawater2060-coder/secureme.git
cd secureme
docker compose up --build
```

Services started:

| Service  | URL / Port            |
|----------|-----------------------|
| Web UI   | http://localhost:3000 |
| API      | http://localhost:8080 |
| Postgres | localhost:5432        |
| Redis    | localhost:6379        |

## Running the Agent Locally (dev)

```bash
export SERVER_URL=http://localhost:8080
export ENROLL_KEY=dev-key          # optional – server accepts any key for now
export DATA_DIR=./agent-data

cd agent
go run ./cmd/agent
```

The agent will enroll itself, then send heartbeats every 30 s and poll for commands every 30 s.

## Accessing the Web UI

Open http://localhost:3000 — the **Devices** table lists every enrolled agent with hostname, OS, IP, enrolled time, and last heartbeat.

## Environment Variables

### Server

| Variable      | Default   | Description                                        |
|---------------|-----------|----------------------------------------------------|
| `LISTEN_ADDR` | `:8080`   | TCP address the HTTP server binds to               |
| `DB_URL`      | _(unset)_ | PostgreSQL DSN. Falls back to in-memory if not set |

### Agent

| Variable     | Default                 | Description                               |
|--------------|-------------------------|-------------------------------------------|
| `SERVER_URL` | `http://localhost:8080` | Base URL of the SecureMe server           |
| `ENROLL_KEY` | _(unset)_               | Pre-shared enrolment key                  |
| `DATA_DIR`   | `./data`                | Directory where `state.json` is persisted |

### Web (build-time)

| Variable       | Default | Description                                            |
|----------------|---------|--------------------------------------------------------|
| `VITE_API_URL` | `''`    | API base URL used by the browser (empty = same origin) |
