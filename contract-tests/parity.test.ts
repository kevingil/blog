import { afterAll, describe, expect, test } from "bun:test";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { isDeepStrictEqual } from "node:util";

const goBase = process.env.GO_BACKEND_BASE_URL;
const rustBase = process.env.RUST_BACKEND_BASE_URL;
const reportPath = process.env.REPORT_PATH;

if (!goBase || !rustBase || !reportPath) {
  throw new Error("GO_BACKEND_BASE_URL, RUST_BACKEND_BASE_URL, and REPORT_PATH are required");
}

type CaseResult = {
  id: string;
  method: string;
  path: string;
  goStatus: number;
  rustStatus: number;
  passed: boolean;
};

const results: CaseResult[] = [];
const contracts = (await readFile("/workspace/docs/porting/CONTRACTS.tsv", "utf8"))
  .trimEnd()
  .split("\n")
  .slice(1)
  .map((line) => line.split("\t"));

const concretePath = (path: string) =>
  path.replace(/\{name\}/g, "crawl").replace(
    /\{[^}]+\}/g,
    "00000000-0000-0000-0000-000000000001",
  );

const publicReadCases = contracts.filter((row) => {
  const [id, , method, path, auth, , , , , , classification] = row;
  return (
    id.startsWith("HTTP-") &&
    classification === "active-contract" &&
    auth === "public" &&
    method === "GET" &&
    !["/", "/health", "/swagger/*", "/api/openapi.json"].includes(path)
  );
});

const publicReadPath = (id: string, path: string) => {
  const resolved = concretePath(path);
  return id === "HTTP-011" ? `${resolved}?query=missing` : resolved;
};

const nextMessage = (socket: WebSocket) =>
  new Promise<unknown>((resolve, reject) => {
    const timeout = setTimeout(() => reject(new Error("timed out waiting for WebSocket frame")), 5_000);
    socket.addEventListener(
      "message",
      (event) => {
        clearTimeout(timeout);
        resolve(JSON.parse(String(event.data)));
      },
      { once: true },
    );
  });

const openSocket = (base: string) =>
  new Promise<WebSocket>((resolve, reject) => {
    const socket = new WebSocket(base.replace(/^http/, "ws") + "/websocket");
    socket.addEventListener("open", () => resolve(socket), { once: true });
    socket.addEventListener("error", () => reject(new Error(`failed to open ${base} WebSocket`)), {
      once: true,
    });
  });

afterAll(async () => {
  await mkdir(reportPath.slice(0, reportPath.lastIndexOf("/")), { recursive: true });
  await writeFile(
    reportPath,
    `${JSON.stringify(
      {
        generatedAt: new Date().toISOString(),
        goBase,
        rustBase,
        cases: results,
        passed: results.every((result) => result.passed),
      },
      null,
      2,
    )}\n`,
  );
});

describe("language-neutral Go/Rust parity boundary", () => {
  test("root and health responses match", async () => {
    for (const path of ["/", "/health"]) {
      const [goResponse, rustResponse] = await Promise.all([
        fetch(`${goBase}${path}`),
        fetch(`${rustBase}${path}`),
      ]);
      const [goBody, rustBody] = await Promise.all([goResponse.json(), rustResponse.json()]);
      expect(rustResponse.status).toBe(goResponse.status);
      expect(rustBody).toEqual(goBody);
      results.push({
        id: path === "/" ? "HTTP-001" : "HTTP-002",
        method: "GET",
        path,
        goStatus: goResponse.status,
        rustStatus: rustResponse.status,
        passed: true,
      });
    }
  });

  for (const row of publicReadCases) {
    const [id, , method, path] = row;
    test(`${id} public read response matches`, async () => {
      const resolvedPath = publicReadPath(id, path);
      const [goResponse, rustResponse] = await Promise.all([
        fetch(`${goBase}${resolvedPath}`),
        fetch(`${rustBase}${resolvedPath}`),
      ]);
      const [goBody, rustBody] = await Promise.all([goResponse.json(), rustResponse.json()]);
      const passed = goResponse.status === rustResponse.status && isDeepStrictEqual(goBody, rustBody);
      results.push({
        id,
        method,
        path,
        goStatus: goResponse.status,
        rustStatus: rustResponse.status,
        passed,
      });
      expect(rustResponse.status).toBe(goResponse.status);
      expect(rustBody).toEqual(goBody);
    });
  }

  for (const row of contracts) {
    const [id, , method, path, auth, , , , , , classification] = row;
    if (
      !id.startsWith("HTTP-") ||
      classification !== "active-contract" ||
      !auth.startsWith("Bearer JWT")
    ) {
      continue;
    }

    test(`${id} rejects anonymous requests in both implementations`, async () => {
      const resolvedPath = concretePath(path);
      const init: RequestInit = { method };
      if (["POST", "PUT", "PATCH"].includes(method)) {
        init.headers = { "content-type": "application/json" };
        init.body = "{}";
      }
      const [goResponse, rustResponse] = await Promise.all([
        fetch(`${goBase}${resolvedPath}`, init),
        fetch(`${rustBase}${resolvedPath}`, init),
      ]);
      const passed = goResponse.status === 401 && rustResponse.status === 401;
      results.push({
        id,
        method,
        path,
        goStatus: goResponse.status,
        rustStatus: rustResponse.status,
        passed,
      });
      expect(goResponse.status).toBe(401);
      expect(rustResponse.status).toBe(401);
    });
  }

  test("WS-001, WS-002, and WS-003 preserve upgrade and request-missing frames", async () => {
    const [goSocket, rustSocket] = await Promise.all([openSocket(goBase), openSocket(rustBase)]);
    const request = JSON.stringify({
      action: "subscribe",
      requestId: "00000000-0000-0000-0000-000000000001",
      channel: "",
    });
    const goMessage = nextMessage(goSocket);
    const rustMessage = nextMessage(rustSocket);
    goSocket.send(request);
    rustSocket.send(request);
    const [goFrame, rustFrame] = await Promise.all([goMessage, rustMessage]);
    goSocket.close();
    rustSocket.close();
    const passed = isDeepStrictEqual(goFrame, rustFrame);
    results.push({
      id: "WS-003",
      method: "GET",
      path: "/websocket",
      goStatus: 101,
      rustStatus: 101,
      passed,
    });
    expect(rustFrame).toEqual(goFrame);
    expect(rustFrame).toEqual({
      requestId: "00000000-0000-0000-0000-000000000001",
      type: "error",
      error: "Request not found",
      done: true,
    });
  });
});
