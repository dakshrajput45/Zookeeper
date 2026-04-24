# app-servers

Node.js app nodes for testing the ZooKeeper coordinator.

## Endpoints exposed by each node
- `POST /vote-request`
- `POST /replication/append`
- `GET /internal/read?key=...`
- `GET /health`
- `POST /admin/heartbeat/stop` (simulate missed heartbeats)
- `POST /admin/heartbeat/start`
- `POST /admin/shutdown` (simulate node crash)

## Run
From `D:\CodeEra\go\app-servers`:

```bash
npm install
npm run start:3
```

This launches:
- `node-1` on `9001`
- `node-2` on `9002`
- `node-3` on `9003`

All nodes auto-register and heartbeat to `http://localhost:8080` by default.

To override ZooKeeper URL:

```bash
ZK_URL=http://localhost:8080 npm run start:3
```

## Simulate node failure
Stop heartbeat for one node:
```bash
curl -X POST http://localhost:9002/admin/heartbeat/stop
```

Bring heartbeat back:
```bash
curl -X POST http://localhost:9002/admin/heartbeat/start
```

Crash one node process:
```bash
curl -X POST http://localhost:9002/admin/shutdown
```

