#!/usr/bin/env node
// Drives or checks the MCP demo in scripts/mcp-demo-prompt.txt.
// Usage:
//   node scripts/mcp-demo.mjs client --base URL --token TOKEN --prompt FILE --log FILE
//   node scripts/mcp-demo.mjs verify --base URL --token TOKEN --trace FILE --mode client|codex
//   node scripts/mcp-demo.mjs follow
// follow reads Codex JSONL on stdin and prints a short line per event.

import { readFileSync, writeFileSync } from "node:fs";
import { createInterface } from "node:readline";

const SEARCHES = [
  "draft",
  "updatePage",
  "upsertProject",
  "upsertArticleSources",
  "versions",
  "listStorageFiles",
];

const command = process.argv[2];
const args = parseArgs(process.argv.slice(3));

try {
  if (command === "client") {
    await runClient(args);
  } else if (command === "verify") {
    await runVerify(args);
  } else if (command === "follow") {
    await runFollow();
  } else {
    throw new Error(
      "usage: mcp-demo.mjs client|verify|follow",
    );
  }
} catch (error) {
  console.error(error instanceof Error ? error.message : error);
  process.exit(1);
}

function parseArgs(argv) {
  const parsed = {};
  for (let i = 0; i < argv.length; i += 1) {
    const key = argv[i];
    if (!key.startsWith("--")) {
      throw new Error(`unexpected argument ${key}`);
    }
    const value = argv[i + 1];
    if (!value || value.startsWith("--")) {
      throw new Error(`${key} needs a value`);
    }
    parsed[key.slice(2)] = value;
    i += 1;
  }
  return parsed;
}

function requireArg(parsed, name) {
  const value = parsed[name];
  if (!value) {
    throw new Error(`--${name} is required`);
  }
  return value;
}

function loadSteps(promptPath) {
  const text = readFileSync(promptPath, "utf8");
  const parts = text.split(/^### Execute: /m).slice(1);
  const steps = parts.map((part) => {
    const name = part.slice(0, part.indexOf("\n")).trim();
    const fence = part.match(/```js\n([\s\S]*?)\n```/);
    if (!fence) {
      throw new Error(`execute block ${name} has no javascript`);
    }
    return { name, code: fence[1] };
  });
  const names = steps.map((step) => step.name);
  const expected = ["drafts", "edit", "page", "project", "sources", "storage", "refuse"];
  if (names.join(",") !== expected.join(",")) {
    throw new Error(`prompt execute blocks are ${names.join(", ")}`);
  }
  for (const query of SEARCHES) {
    if (!text.includes(query)) {
      throw new Error(`prompt is missing search query ${query}`);
    }
  }
  return steps;
}

async function mcpCall(base, token, message) {
  const response = await fetch(`${base}/mcp`, {
    method: "POST",
    headers: {
      authorization: `Bearer ${token}`,
      "content-type": "application/json",
      accept: "application/json, text/event-stream",
    },
    body: JSON.stringify(message),
  });
  const text = await response.text();
  if (!response.ok) {
    throw new Error(`MCP HTTP ${response.status}: ${text.slice(0, 400)}`);
  }
  if (!Object.hasOwn(message, "id")) {
    return null;
  }
  const payload = parseMcpBody(text, response.headers.get("content-type") || "");
  if (payload.error) {
    throw new Error(`MCP ${message.method} error: ${JSON.stringify(payload.error)}`);
  }
  return payload.result;
}

function parseMcpBody(text, contentType) {
  if (contentType.includes("text/event-stream")) {
    const data = text
      .split("\n")
      .filter((line) => line.startsWith("data:"))
      .map((line) => line.slice(5).trim())
      .filter((line) => line && line !== "[DONE]");
    const last = data[data.length - 1];
    if (!last) {
      throw new Error("MCP event stream had no data");
    }
    return JSON.parse(last);
  }
  return JSON.parse(text);
}

function toolOutcome(result) {
  const text = result?.content?.[0]?.text ?? "";
  return { isError: Boolean(result?.isError), text };
}

async function runClient(args) {
  const base = requireArg(args, "base").replace(/\/$/, "");
  const token = requireArg(args, "token");
  const prompt = requireArg(args, "prompt");
  const logPath = requireArg(args, "log");
  const steps = loadSteps(prompt);
  let id = 1;

  async function tool(name, toolArgs) {
    await mcpCall(base, token, {
      jsonrpc: "2.0",
      id: id,
      method: "initialize",
      params: {
        protocolVersion: "2025-11-25",
        capabilities: {},
        clientInfo: { name: "mcp-demo", version: "1" },
      },
    });
    id += 1;
    const result = await mcpCall(base, token, {
      jsonrpc: "2.0",
      id: id,
      method: "tools/call",
      params: { name, arguments: toolArgs },
    });
    id += 1;
    return toolOutcome(result);
  }

  const searches = [];
  for (const query of SEARCHES) {
    const outcome = await tool("search", { query, limit: 25 });
    if (outcome.isError) {
      throw new Error(`search ${query} failed: ${outcome.text}`);
    }
    const operations = JSON.parse(outcome.text).operations ?? [];
    const brief = operations.map(
      (operation) => `${operation.method} ${operation.path} ${operation.operationId}`,
    );
    console.log(`search ${query}: ${brief.slice(0, 8).join("; ")}`);
    searches.push({ query, operations: brief });
  }

  const executed = [];
  for (const step of steps) {
    const outcome = await tool("execute", { code: step.code });
    const line = outcome.text.length > 500 ? `${outcome.text.slice(0, 500)}...` : outcome.text;
    console.log(`execute ${step.name}: ${outcome.isError ? "error" : "ok"} ${line}`);
    executed.push({ name: step.name, isError: outcome.isError, text: outcome.text });
    if (step.name === "refuse") {
      if (!outcome.isError || !/refused/i.test(outcome.text) || !outcome.text.includes("/auth/me")) {
        throw new Error(`expected /auth/me to be refused, got: ${outcome.text}`);
      }
      continue;
    }
    if (outcome.isError) {
      throw new Error(`execute ${step.name} failed: ${outcome.text}`);
    }
  }

  const project = executed.find((step) => step.name === "project");
  if (!project || !/"same"\s*:\s*true/.test(project.text)) {
    throw new Error("project upsert did not report the same id twice");
  }
  const storage = executed.find((step) => step.name === "storage");
  if (!storage || !/"storageStatus"\s*:\s*200/.test(storage.text)) {
    throw new Error("storage listing did not return status 200");
  }

  writeFileSync(
    logPath,
    JSON.stringify({ searches, executed }, null, 2),
  );
  console.log(`wrote ${logPath}`);
}

async function runVerify(args) {
  const base = requireArg(args, "base").replace(/\/$/, "");
  const token = requireArg(args, "token");
  const trace = requireArg(args, "trace");
  const mode = requireArg(args, "mode");
  const failures = [];

  const articles = await api(base, token, "/blog/articles?status=drafts&articlesPerPage=100&page=1");
  const items = articles.body?.data?.articles ?? [];
  const harbor = items.find((item) => item.article?.draft_title === "Harbor Notes, revised");
  const night = items.find((item) => item.article?.draft_title === "Night Shift");
  if (!harbor) {
    failures.push("missing draft Harbor Notes, revised");
  } else {
    if (harbor.article.published_at) {
      failures.push("Harbor Notes, revised is published");
    }
    const tags = (harbor.tags ?? []).map((tag) => tag.name);
    if (!tags.includes("harbor") || !tags.includes("revised")) {
      failures.push(`Harbor Notes tags are ${tags.join(", ")}`);
    }
    if (!String(harbor.article.draft_content).includes("Revised after the first draft.")) {
      failures.push("Harbor Notes body is missing the revision");
    }
    const versions = await api(
      base,
      token,
      `/blog/articles/${harbor.article.slug}/versions`,
    );
    const history = versions.body?.data?.versions ?? [];
    const prior = history.some(
      (version) => version.title === "Harbor Notes" || String(version.content).includes("cormorants"),
    );
    if (versions.status !== 200 || history.length < 1 || !prior) {
      failures.push("Harbor Notes has no pre-edit version");
    } else {
      console.log(`versions: ${history.length} for ${harbor.article.slug}`);
    }
  }
  if (!night) {
    failures.push("missing draft Night Shift");
  } else {
    if (night.article.published_at) {
      failures.push("Night Shift is published");
    }
    const tags = (night.tags ?? []).map((tag) => tag.name);
    if (!tags.includes("night") || !tags.includes("shift")) {
      failures.push(`Night Shift tags are ${tags.join(", ")}`);
    }
    if (!String(night.article.draft_content).includes("The night shift starts")) {
      failures.push("Night Shift body is missing");
    }
  }

  const pages = await api(base, token, "/dashboard/pages?page=1&perPage=100");
  const page = (pages.body?.data?.pages ?? []).find((item) => item.slug === "mcp-demo-desk");
  if (!page || page.title !== "Desk, revised" || !String(page.content).includes("Revised after the first save.")) {
    failures.push("page mcp-demo-desk was not revised");
  }

  const projects = await collectProjects(base, token);
  const harborProjects = projects.filter((project) => project.url === "https://example.test/mcp-demo-harbor");
  if (harborProjects.length !== 1 || harborProjects[0].title !== "Harbor project, second write") {
    failures.push(
      `expected one harbor project, found ${harborProjects.length}: ${harborProjects.map((project) => project.title).join(", ")}`,
    );
  }

  if (harbor) {
    const sources = await api(base, token, `/sources/article/${harbor.article.id}`);
    const urls = (sources.body?.data?.sources ?? []).map((source) => source.url);
    for (const url of ["https://example.test/mcp-demo-tide", "https://example.test/mcp-demo-log"]) {
      if (!urls.includes(url)) {
        failures.push(`missing source ${url}`);
      }
    }
    if (sources.status === 200 && urls.length) {
      console.log(`sources: ${urls.length}`);
    }
  }

  const storage = await api(base, token, "/storage/files");
  if (storage.status !== 200 || !Array.isArray(storage.body?.data?.files)) {
    failures.push(`storage list returned ${storage.status}`);
  }

  checkTrace(trace, mode, failures);

  if (failures.length) {
    for (const failure of failures) {
      console.error(`fail: ${failure}`);
    }
    throw new Error(`${failures.length} verification check(s) failed`);
  }

  console.log("verified drafts: Harbor Notes, revised; Night Shift");
  console.log("verified page: Desk, revised");
  console.log("verified project: Harbor project, second write (one row)");
  console.log("verified sources and a refused /auth/me call");
}

function checkTrace(trace, mode, failures) {
  const text = readFileSync(trace, "utf8");
  if (mode === "client") {
    const log = JSON.parse(text);
    const queries = (log.searches ?? []).map((search) => search.query);
    for (const query of SEARCHES) {
      if (!queries.includes(query)) {
        failures.push(`client log missing search ${query}`);
      }
    }
    const byName = new Map((log.executed ?? []).map((step) => [step.name, step]));
    for (const name of ["drafts", "edit", "page", "project", "sources", "storage"]) {
      const step = byName.get(name);
      if (!step || step.isError) {
        failures.push(`client log missing successful execute ${name}`);
      }
    }
    const refused = byName.get("refuse");
    if (!refused?.isError || !/refused/i.test(refused.text) || !refused.text.includes("/auth/me")) {
      failures.push("client log does not show /auth/me refused");
    }
    return;
  }
  if (mode !== "codex") {
    failures.push(`unknown trace mode ${mode}`);
    return;
  }
  if (!text.trim()) {
    failures.push("Codex event log is empty");
    return;
  }
  // Tool results are JSON strings inside the JSONL events, so quotes are escaped.
  const readable = text.replace(/\\"/g, '"');
  for (const query of SEARCHES) {
    if (!readable.includes(query)) {
      failures.push(`Codex transcript missing search query ${query}`);
    }
  }
  if (!/"same"\s*:\s*true/.test(readable)) {
    failures.push("Codex transcript does not show the project upsert returning the same id");
  }
  if (!/"storageStatus"\s*:\s*200/.test(readable)) {
    failures.push("Codex transcript does not show storage status 200");
  }
  if (!/refused/i.test(readable) || !readable.includes("/auth/me")) {
    failures.push("Codex transcript does not show /auth/me refused");
  }
}

async function collectProjects(base, token) {
  const projects = [];
  for (let page = 1; page <= 20; page += 1) {
    const response = await api(base, token, `/projects?page=${page}&perPage=100`);
    const batch = response.body?.data?.projects ?? [];
    projects.push(...batch);
    const total = response.body?.data?.total ?? projects.length;
    if (projects.length >= total || batch.length === 0) {
      break;
    }
  }
  return projects;
}

async function api(base, token, path) {
  const response = await fetch(`${base}${path}`, {
    headers: { authorization: `Bearer ${token}`, accept: "application/json" },
  });
  const text = await response.text();
  let body = null;
  if (text) {
    try {
      body = JSON.parse(text);
    } catch {
      body = text;
    }
  }
  return { status: response.status, body };
}

async function runFollow() {
  const lines = createInterface({ input: process.stdin, crlfDelay: Infinity });
  for await (const line of lines) {
    if (!line.trim()) {
      continue;
    }
    let event;
    try {
      event = JSON.parse(line);
    } catch {
      console.log(line);
      continue;
    }
    const summary = summarize(event);
    if (summary) {
      console.log(summary);
    }
  }
}

function summarize(event) {
  const type = event.type || "event";
  if (type === "thread.started") {
    return "codex thread started";
  }
  if (type === "turn.started") {
    return "turn started";
  }
  if (type === "turn.completed") {
    return "turn completed";
  }
  if (type === "turn.failed" || type === "error") {
    return `${type}: ${JSON.stringify(event).slice(0, 400)}`;
  }
  if (type === "item.completed" || type === "item.started") {
    const item = event.item || {};
    const itemType = item.type || type;
    if (itemType.includes("mcp") || itemType === "tool_call") {
      const name = item.tool || item.name || item.tool_name || "";
      const server = item.server || item.server_name || "";
      const status = item.status || type;
      return `mcp ${server} ${name} (${status})`.replace(/\s+/g, " ").trim();
    }
    if (itemType === "agent_message" || itemType === "agent_message_item") {
      const message = item.text || item.message || "";
      return message ? `codex: ${message}` : null;
    }
    if (itemType === "command_execution" || itemType === "command") {
      return `command ${item.command || ""} (${item.status || type})`;
    }
    if (itemType === "reasoning") {
      return null;
    }
    return `${itemType} ${type}`;
  }
  return null;
}
