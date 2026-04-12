/**
 * Stdio-to-HTTP MCP proxy for task-tree-service-v2.
 * Reads JSON-RPC from stdin, forwards to http://127.0.0.1:8880/mcp,
 * writes responses to stdout.
 */

import { createInterface } from "node:readline";
import http from "node:http";

const TARGET = "http://127.0.0.1:8880/mcp";
let sessionId = "";
let pending = 0;
let inputClosed = false;

function maybeExit() {
  if (inputClosed && pending === 0) process.exit(0);
}

function post(body) {
  return new Promise((resolve, reject) => {
    const url = new URL(TARGET);
    const headers = {
      "Content-Type": "application/json",
      Accept: "application/json",
    };
    if (sessionId) {
      headers["Mcp-Session-Id"] = sessionId;
    }
    const req = http.request(
      { hostname: url.hostname, port: url.port, path: url.pathname, method: "POST", headers },
      (res) => {
        const chunks = [];
        res.on("data", (c) => chunks.push(c));
        res.on("end", () => {
          const sid = res.headers["mcp-session-id"];
          if (sid) sessionId = sid;
          resolve({ status: res.statusCode, body: Buffer.concat(chunks).toString() });
        });
      }
    );
    req.on("error", reject);
    req.end(body);
  });
}

const rl = createInterface({ input: process.stdin, terminal: false });

rl.on("line", async (line) => {
  const trimmed = line.trim();
  if (!trimmed) return;

  pending++;
  try {
    JSON.parse(trimmed); // validate
    const { status, body } = await post(trimmed);

    if (status !== 202 && body.trim()) {
      process.stdout.write(body + "\n");
    }
  } catch (err) {
    try {
      const parsed = JSON.parse(trimmed);
      if (parsed.id !== undefined) {
        const errResp = JSON.stringify({
          jsonrpc: "2.0",
          id: parsed.id,
          error: { code: -32603, message: err.message },
        });
        process.stdout.write(errResp + "\n");
      }
    } catch {
      // ignore
    }
  } finally {
    pending--;
    maybeExit();
  }
});

rl.on("close", () => {
  inputClosed = true;
  maybeExit();
});
