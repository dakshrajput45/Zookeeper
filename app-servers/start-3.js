const { spawn } = require("child_process");
const path = require("path");

const script = path.join(__dirname, "server.js");
const zkUrl = process.env.ZK_URL || "http://localhost:8080";

const nodes = [
  { NODE_ID: "node-1", PORT: "9001" },
  { NODE_ID: "node-2", PORT: "9002" },
  { NODE_ID: "node-3", PORT: "9003" }
];

const children = nodes.map((node) =>
  spawn(process.execPath, [script], {
    stdio: "inherit",
    env: {
      ...process.env,
      ZK_URL: zkUrl,
      ...node
    }
  })
);

function shutdown() {
  for (const child of children) {
    child.kill("SIGINT");
  }
}

process.on("SIGINT", shutdown);
process.on("SIGTERM", shutdown);

