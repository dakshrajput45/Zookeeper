# ZooKeeper MVP Test Plan

## 1. Start services

### 1.1 Start ZooKeeper coordinator
```bash
cd D:\CodeEra\go\zookeeper
go run ./cmd/zkd
```

### 1.2 Start Node app servers
```bash
cd D:\CodeEra\go\app-servers
npm install
npm run start:3
```

## 2. Baseline checks

```bash
curl -s http://localhost:8080/health
curl -s http://localhost:8080/nodes/alive
curl -s http://localhost:8080/leader
curl -s http://localhost:8080/election/state
```

Expected:
- 3 nodes visible
- 1 leader present

## 3. Write + replication test

```bash
curl -s -X POST http://localhost:8080/write \
  -H "Content-Type: application/json" \
  -d '{"key":"k1","value":"v1"}'

curl -s http://localhost:8080/replication/state
```

Expected:
- write returns committed when quorum reached
- replication state contains new entry with `acked_by` and `committed` status

## 4. Leader-routed read test

```bash
curl -s "http://localhost:8080/read?key=k1"
```

Expected:
- `found: true`
- `value: v1`

## 5. Failover test (heartbeat stop)

### 5.1 Find current leader
```bash
curl -s http://localhost:8080/leader
```

### 5.2 Stop that leader heartbeat
```bash
curl -X POST http://localhost:<leader-port>/admin/heartbeat/stop
```

### 5.3 Wait > 15s, then check state
```bash
curl -s http://localhost:8080/leader
curl -s http://localhost:8080/election/state
curl -s http://localhost:8080/nodes/alive
```

Expected:
- leader changes to another node
- old leader shows `is_alive: false`

## 6. Crash test

```bash
curl -X POST http://localhost:9002/admin/shutdown
curl -s http://localhost:8080/nodes/alive
```

Expected:
- node-2 becomes dead after timeout
- cluster still elects leader if quorum voters remain

## 7. Recovery test

Restart stopped/crashed node and verify:
```bash
curl -X POST http://localhost:9002/admin/heartbeat/start
curl -s http://localhost:8080/nodes/alive
```

Expected:
- node appears alive again after heartbeat

## 8. Negative tests

### 8.1 Invalid write payload
```bash
curl -i -X POST http://localhost:8080/write \
  -H "Content-Type: application/json" \
  -d '{"key":"","value":"x"}'
```

Expected:
- `400 Bad Request`

### 8.2 Invalid read payload
```bash
curl -i "http://localhost:8080/read?key="
```

Expected:
- `400 Bad Request`

## 9. Pass criteria

- Registration and heartbeat stable for all nodes
- Leader exists under normal conditions
- Leader failover happens automatically after timeout
- Writes commit only with quorum ACK
- Reads return expected value through leader-routed path
