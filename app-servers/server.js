const express = require("express");

const app = express();
app.use(express.json());

const NODE_ID = process.env.NODE_ID || "node-1";
const PORT = Number(process.env.PORT || 9001);
const HOST = process.env.HOST || "localhost";
const ZK_URL = (process.env.ZK_URL || "http://localhost:8080").replace(/\/$/, "");
const HEARTBEAT_MS = Number(process.env.HEARTBEAT_MS || 3000);

const store = new Map();
let lastIndex = 0;
let lastTerm = 0;
let heartbeatEnabled = true;

function serverAddress() {
  return `http://${HOST}:${PORT}`;
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function hash(input) {
  let h = 0;
  for (let i = 0; i < input.length; i += 1) {
    h = (h * 31 + input.charCodeAt(i)) >>> 0;
  }
  return h;
}

async function registerWithZookeeper() {
  const payload = {
    node_id: NODE_ID,
    address: serverAddress()
  };

  const resp = await fetch(`${ZK_URL}/nodes/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });

  if (!resp.ok) {
    throw new Error(`register failed: ${resp.status}`);
  }
}

async function sendHeartbeat() {
  const payload = { node_id: NODE_ID };
  const resp = await fetch(`${ZK_URL}/nodes/heartbeat`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });
  if (!resp.ok) {
    throw new Error(`heartbeat failed: ${resp.status}`);
  }
}

async function heartbeatLoop() {
  while (true) {
    if (heartbeatEnabled) {
      try {
        await sendHeartbeat();
      } catch (err) {
        console.error(`[${NODE_ID}] heartbeat error:`, err.message);
      }
    }
    await sleep(HEARTBEAT_MS);
  }
}

app.get("/health", (_, res) => {
  res.json({
    status: "ok",
    node_id: NODE_ID,
    port: PORT,
    last_index: lastIndex,
    last_term: lastTerm,
    heartbeat_enabled: heartbeatEnabled
  });
});

app.post("/admin/heartbeat/stop", (_, res) => {
  heartbeatEnabled = false;
  res.json({ ok: true, node_id: NODE_ID, heartbeat_enabled: heartbeatEnabled });
});

app.post("/admin/heartbeat/start", (_, res) => {
  heartbeatEnabled = true;
  res.json({ ok: true, node_id: NODE_ID, heartbeat_enabled: heartbeatEnabled });
});

app.post("/admin/shutdown", (_, res) => {
  res.json({ ok: true, node_id: NODE_ID, shutting_down: true });
  setTimeout(() => process.exit(0), 100);
});

app.post("/vote-request", (req, res) => {
  const { term, candidate_ids: candidateIds } = req.body || {};
  if (!Array.isArray(candidateIds) || candidateIds.length === 0) {
    return res.status(400).json({ error: "candidate_ids is required" });
  }

  const idx = hash(`${NODE_ID}:${term || 0}`) % candidateIds.length;
  return res.json({ voted_for: candidateIds[idx] });
});

app.post("/replication/append", (req, res) => {
  const { index, term, key, value } = req.body || {};
  if (!key && key !== "") {
    return res.status(400).json({ error: "key is required" });
  }

  if (typeof index === "number") {
    lastIndex = Math.max(lastIndex, index);
  }
  if (typeof term === "number") {
    lastTerm = Math.max(lastTerm, term);
  }
  store.set(String(key), String(value ?? ""));

  return res.json({ ok: true, node_id: NODE_ID });
});

app.get("/internal/read", (req, res) => {
  const key = String(req.query.key || "").trim();
  if (!key) {
    return res.status(400).json({ error: "key is required" });
  }

  if (!store.has(key)) {
    return res.json({ found: false, value: "" });
  }

  return res.json({ found: true, value: store.get(key) });
});

app.listen(PORT, async () => {
  console.log(`[${NODE_ID}] listening on ${serverAddress()}`);
  try {
    await registerWithZookeeper();
    console.log(`[${NODE_ID}] registered to ${ZK_URL}`);
  } catch (err) {
    console.error(`[${NODE_ID}] initial register failed:`, err.message);
  }
  heartbeatLoop();
});

