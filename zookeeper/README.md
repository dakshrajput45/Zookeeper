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
- automatic failover trigger via heartbeat-deadline based election monitor

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
- `GET /election/state` (debug only; does not trigger election)
- `POST /write` (client write entrypoint; no node id required)
- `GET /read?key=...` (leader-routed read)
- `GET /replication/state` (debug replication log in coordinator)

## Automatic election behavior
- The coordinator does not poll on a fixed interval for failover.
- It tracks leader heartbeat deadline: `last_leader_heartbeat + heartbeat_timeout`.
- If no heartbeat arrives before deadline, election starts automatically.
- Election requests are sent to app nodes via `POST /vote-request`.
- Current candidate set defaults to all alive nodes.
- Leader is selected by highest vote count.
- If votes tie, candidate with smallest `node_id` wins.

## Current MVP election notes
- Voting decision is delegated to app nodes via `/vote-request`.
- Election state is observable via `GET /election/state`.
- This is an MVP election model and will be hardened further in later phases.

## Write replication behavior (coordinator role)
- Client calls `POST /write` with `{ "key": "...", "value": "..." }`.
- Coordinator resolves current leader internally.
- Coordinator sends append request to every alive node at `POST /replication/append`.
- Commit succeeds only when ACK count reaches quorum (`(alive_nodes/2)+1`).
- If quorum is not reached, coordinator returns a quorum failure and write stays uncommitted.
- Reads use simple leader approach: coordinator forwards `GET /read?key=...` to leader's internal read API.

## App node contract (required)
Each registered app node must expose:
- `POST /vote-request`
- `POST /replication/append`
- `GET /internal/read?key=...`

Request body sent by coordinator:
```json
{
  "term": 2,
  "candidate_ids": ["node-1", "node-2", "node-3"]
}
```

Expected app response:
```json
{
  "voted_for": "node-2"
}
```

Replication append request sent by coordinator:
```json
{
  "index": 11,
  "term": 3,
  "leader_id": "node-2",
  "key": "k1",
  "value": "v1"
}
```

Expected replication append response:
- HTTP `200 OK` for ACK

Expected internal read response:
```json
{
  "found": true,
  "value": "v1"
}
```

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
curl -s http://localhost:8080/election/state
curl -s -X POST http://localhost:8080/write -H "Content-Type: application/json" -d '{"key":"k1","value":"v1"}'
curl -s "http://localhost:8080/read?key=k1"
curl -s http://localhost:8080/replication/state
```

