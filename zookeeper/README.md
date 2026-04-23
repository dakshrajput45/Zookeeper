# zookeeper

ZooKeeper-like coordination service MVP in Go.

## Setup
### Prerequisites
- Go 1.26+
- Git

### Install dependencies
From repository root:

```bash
go mod tidy
```

## Current runnable baseline
Layered HTTP setup with:
- `router -> controller -> service`
- coordinator APIs for node registration, heartbeat, leader lookup, and alive nodes

## Project layout
- `cmd\zkd` - application entrypoint
- `internal\app` - server wiring and bootstrap
- `internal\router` - route registration
- `internal\controller` - HTTP handlers
- `internal\service` - business logic layer
- `proto` - protobuf definitions for upcoming gRPC phase

## Run
From repository root:

```bash
go run ./cmd/zkd
```

## Publish to GitHub (`zookeeper`)
From repository root:

```bash
git init
git add .
git commit -m "Initial commit"
git branch -M main
git remote add origin https://github.com/<your-username>/zookeeper.git
git push -u origin main
```

## Endpoints
- `GET /health`
- `POST /nodes/register`
- `POST /nodes/heartbeat`
- `GET /leader`
- `GET /nodes/alive`

## Quick curl examples
```bash
curl -s http://localhost:8080/health

curl -s -X POST http://localhost:8080/nodes/register \
  -H "Content-Type: application/json" \
  -d '{"node_id":"node-1","address":"http://localhost:9001"}'

curl -s -X POST http://localhost:8080/nodes/heartbeat \
  -H "Content-Type: application/json" \
  -d '{"node_id":"node-1"}'

curl -s http://localhost:8080/leader
curl -s http://localhost:8080/nodes/alive
```

