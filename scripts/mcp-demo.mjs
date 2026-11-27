#!/usr/bin/env node
// Drives or checks the MCP demo in scripts/mcp-demo-prompt.txt.
// Usage:
//   node scripts/mcp-demo.mjs client --base URL --token TOKEN --log FILE
//   node scripts/mcp-demo.mjs verify --base URL --token TOKEN --trace FILE --mode client|codex
//   node scripts/mcp-demo.mjs follow
// follow reads Codex JSONL on stdin and prints a short line per event.

import { readFileSync, writeFileSync } from "node:fs";
import { createInterface } from "node:readline";

const PAGE_SLUG = "mcp-demo-desk";
const PROJECT_URL = "https://example.test/mcp-demo-harbor";

// Fallback path used only when Codex is not run. The demo prompt has no
// snippets; this client searches, then sends its own code.
const CLIENT_SEARCHES = ["draft", "page", "project", "source", "storage"];
const CLIENT_STEPS = [
  {
    name: "drafts",
    code: `const specs = [
  { title: "Harbor Notes", tags: ["harbor", "notes"], markdown: "The 6am ferry leaves before the bakery opens. I counted twelve cormorants on the breakwater and wrote the tide as 1.4 meters, falling.\\n\\nThis draft is the place I keep the numbers that do not belong in the logbook." },
  { title: "Night Shift", tags: ["night", "shift"], markdown: "The night shift starts when the last bus turns around at the station. The radio sits on the shelf above the kettle.\\n\\nI keep a list of trains that left late and the names of the people who asked if the rain would stop." }
];
const created = [];
for (let i = 0; i < specs.length; i++) {
  const spec = specs[i];
  const response = await blog.request({ method: "POST", path: "/blog/drafts", body: { title: spec.title, markdown: spec.markdown, tags: spec.tags } });
  if (response.status !== 201) throw new Error("create draft failed " + response.status + " " + JSON.stringify(response.body));
  created.push(response.body.data.article.draft_title);
}
const listed = await blog.request({ method: "GET", path: "/blog/articles", query: { status: "drafts", articlesPerPage: 100, page: 1 } });
if (listed.status !== 200) throw new Error("list drafts failed " + listed.status);
return { created: created, listed: listed.body.data.articles.map(function (item) { return item.article.draft_title; }) };`,
  },
  {
    name: "edit",
    code: `const listed = await blog.request({ method: "GET", path: "/blog/articles", query: { status: "drafts", articlesPerPage: 100, page: 1 } });
let article = null;
const items = listed.body.data.articles;
for (let i = 0; i < items.length; i++) {
  if (items[i].article.draft_title === "Harbor Notes") article = items[i].article;
}
if (!article) throw new Error("Harbor Notes was not listed");
const updated = await blog.request({
  method: "POST",
  path: "/blog/articles/" + article.slug + "/update",
  body: {
    title: "Harbor Notes, revised",
    content: article.draft_content + "\\n\\nRevised after the first draft. The lamp on the north pier was repaired before noon.",
    tags: ["harbor", "revised"],
    image_url: ""
  }
});
if (updated.status !== 200) throw new Error("update failed " + updated.status + " " + JSON.stringify(updated.body));
const slug = updated.body.data.article.slug;
const versions = await blog.request({ method: "GET", path: "/blog/articles/" + slug + "/versions" });
if (versions.status !== 200) throw new Error("versions failed " + versions.status);
return { title: updated.body.data.article.draft_title, slug: slug, versionCount: versions.body.data.versions.length };`,
  },
  {
    name: "page",
    code: `const created = await blog.request({
  method: "POST",
  path: "/dashboard/pages",
  body: { slug: "mcp-demo-desk", title: "Desk", content: "A short page written through the scripted client, before the revision.", description: "Demo page.", image_url: "", is_published: false }
});
if (created.status !== 201) throw new Error("page create failed " + created.status + " " + JSON.stringify(created.body));
const edited = await blog.request({
  method: "PUT",
  path: "/dashboard/pages/" + created.body.data.id,
  body: { title: "Desk, revised", content: "A short page written through the scripted client. Revised after the first save, with a longer body." }
});
if (edited.status !== 200) throw new Error("page update failed " + edited.status + " " + JSON.stringify(edited.body));
return { slug: edited.body.data.slug, title: edited.body.data.title };`,
  },
  {
    name: "project",
    code: `const url = "https://example.test/mcp-demo-harbor";
const first = await blog.request({
  method: "POST", path: "/projects/upsert",
  body: { title: "Harbor project", description: "First write of the harbor project.", content: "First body.", tags: ["harbor"], image_url: "", url: url }
});
const second = await blog.request({
  method: "POST", path: "/projects/upsert",
  body: { title: "Harbor project, second write", description: "Second write of the harbor project.", content: "Second body, same URL.", tags: ["harbor"], image_url: "", url: url }
});
if (first.status !== 200 || second.status !== 200) throw new Error("upsert failed " + first.status + " " + second.status);
return { firstId: first.body.data.id, secondId: second.body.data.id, same: first.body.data.id === second.body.data.id, title: second.body.data.title };`,
  },
  {
    name: "sources",
    code: `const listed = await blog.request({ method: "GET", path: "/blog/articles", query: { status: "drafts", articlesPerPage: 100, page: 1 } });
let articleId = null;
const items = listed.body.data.articles;
for (let i = 0; i < items.length; i++) {
  if (items[i].article.draft_title === "Harbor Notes, revised") articleId = items[i].article.id;
}
if (!articleId) throw new Error("revised draft was not listed");
const saved = await blog.request({
  method: "PUT",
  path: "/sources/article/" + articleId,
  body: { sources: [
    { title: "Tide chart", content: "Heights for the week.", url: "https://example.test/mcp-demo-tide", source_type: "web" },
    { title: "Logbook", content: "Notes from the dock.", url: "https://example.test/mcp-demo-log", source_type: "web" }
  ] }
});
if (saved.status !== 200) throw new Error("sources failed " + saved.status + " " + JSON.stringify(saved.body));
return { count: saved.body.data.sources.length };`,
  },
  {
    name: "storage",
    code: `const listed = await blog.request({ method: "GET", path: "/storage/files" });
if (listed.status !== 200) throw new Error("storage failed " + listed.status + " " + JSON.stringify(listed.body));
return { storageStatus: listed.status, files: listed.body.data.files.length };`,
  },
  {
    name: "refuse",
    code: `return await blog.request({ method: "GET", path: "/auth/me" });`,
  },
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
  const logPath = requireArg(args, "log");
  const steps = CLIENT_STEPS;
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
  for (const query of CLIENT_SEARCHES) {
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
  const drafts = (articles.body?.data?.articles ?? []).filter((item) => {
    const article = item.article ?? {};
    return !article.published_at && String(article.draft_content ?? "").trim().length >= 80;
  });
  if (drafts.length < 2) {
    failures.push(`expected at least two unpublished drafts with real prose, found ${drafts.length}`);
  }
  const titles = [];
  let revised = null;
  for (const item of drafts) {
    const article = item.article;
    const tags = (item.tags ?? []).map((tag) => tag.name).filter(Boolean);
    if (tags.length < 1) {
      failures.push(`draft ${article.draft_title} has no tags`);
    }
    titles.push(article.draft_title);
    const versions = await api(base, token, `/blog/articles/${article.slug}/versions`);
    const history = versions.body?.data?.versions ?? [];
    const prior = history.find(
      (version) => version.title !== article.draft_title || version.content !== article.draft_content,
    );
    if (versions.status === 200 && prior && !revised) {
      revised = {
        title: article.draft_title,
        slug: article.slug,
        versions: history.length,
        priorTitle: prior.title,
      };
      console.log(`versions: ${history.length} for ${article.slug} (prior title ${prior.title})`);
    }
  }
  if (!revised) {
    failures.push("no draft has a prior version with a different title or body");
  }

  let sourceCount = 0;
  for (const item of drafts) {
    const sources = await api(base, token, `/sources/article/${item.article.id}`);
    const rows = (sources.body?.data?.sources ?? []).filter(
      (source) => source.title && source.url && source.content,
    );
    if (rows.length > sourceCount) {
      sourceCount = rows.length;
    }
  }
  if (sourceCount < 2) {
    failures.push(`expected two sources on an article, found ${sourceCount}`);
  } else {
    console.log(`sources: ${sourceCount}`);
  }

  const pages = await api(base, token, "/dashboard/pages?page=1&perPage=100");
  const page = (pages.body?.data?.pages ?? []).find((item) => item.slug === PAGE_SLUG);
  if (!page) {
    failures.push(`page ${PAGE_SLUG} is missing`);
  } else if (String(page.content ?? "").trim().length < 40) {
    failures.push(`page ${PAGE_SLUG} body is too short`);
  } else if (!page.created_at || page.updated_at === page.created_at) {
    failures.push(`page ${PAGE_SLUG} was not updated after it was created`);
  }

  const projects = await collectProjects(base, token);
  const matches = projects.filter((project) => project.url === PROJECT_URL);
  if (matches.length !== 1 || !matches[0].title || !matches[0].description) {
    failures.push(
      `expected one project at ${PROJECT_URL}, found ${matches.length}: ${matches.map((project) => project.title).join(", ")}`,
    );
  }

  const storage = await api(base, token, "/storage/files");
  if (storage.status !== 200 || !Array.isArray(storage.body?.data?.files)) {
    failures.push(`storage list returned ${storage.status}`);
  }

  checkTrace(trace, mode, failures, [...titles, revised?.priorTitle].filter(Boolean));

  if (failures.length) {
    for (const failure of failures) {
      console.error(`fail: ${failure}`);
    }
    throw new Error(`${failures.length} verification check(s) failed`);
  }

  console.log(`verified drafts: ${titles.join("; ")}`);
  console.log(`verified revision: ${revised.title}`);
  console.log(`verified page: ${page.title} (${page.slug})`);
  console.log(`verified project: ${matches[0].title} (one row)`);
  console.log(`verified sources: ${sourceCount}`);
  console.log("verified a refused current-user call");
}

function checkTrace(trace, mode, failures, titles) {
  const text = readFileSync(trace, "utf8");
  if (mode === "client") {
    const log = JSON.parse(text);
    if ((log.searches ?? []).length < 1) {
      failures.push("client log has no search calls");
    }
    const refused = (log.executed ?? []).find((step) => step.name === "refuse");
    if (!refused?.isError || !/refused/i.test(refused.text)) {
      failures.push("client log does not show a refused current-user call");
    }
    const listed = (log.executed ?? []).some(
      (step) => !step.isError && distinctTitles(step.text, titles).length >= 2,
    );
    if (titles.length >= 2 && !listed) {
      failures.push("client log did not list both drafts");
    }
    return;
  }
  if (mode !== "codex") {
    failures.push(`unknown trace mode ${mode}`);
    return;
  }
  const calls = codexToolCalls(text);
  if (calls.length === 0) {
    failures.push("Codex transcript has no MCP tool calls");
    return;
  }
  const searchAt = calls.findIndex((call) => call.tool === "search" && !call.failed);
  const executeAt = calls.findIndex((call) => call.tool === "execute");
  if (searchAt === -1 || (executeAt !== -1 && searchAt > executeAt)) {
    failures.push("Codex did not search the API before execute");
  }
  const code = calls
    .filter((call) => call.tool === "execute")
    .map((call) => call.code)
    .join("\n");
  if (!code.includes(PAGE_SLUG)) {
    failures.push(`Codex execute code never used the page slug ${PAGE_SLUG}`);
  }
  const urlWrites = code.split(PROJECT_URL).length - 1;
  if (urlWrites < 2) {
    failures.push(`Codex execute code mentions ${PROJECT_URL} ${urlWrites} time(s)`);
  }
  const stored = calls.some(
    (call) => call.tool === "execute" && !call.failed && /storage/i.test(call.code),
  );
  if (!stored) {
    failures.push("Codex has no successful execute that lists storage");
  }
  const refused = calls.some(
    (call) => call.tool === "execute" && call.failed && /refused/i.test(call.text),
  );
  if (!refused) {
    failures.push("Codex transcript does not show a refused current-user call");
  }
  if (titles.length >= 2) {
    const listed = calls.some(
      (call) => call.tool === "execute" && !call.failed && distinctTitles(call.text, titles).length >= 2,
    );
    if (!listed) {
      failures.push("Codex transcript has no execute result that lists both drafts");
    }
  }
}

function distinctTitles(text, titles) {
  const sorted = [...new Set(titles.filter(Boolean))].sort((a, b) => b.length - a.length);
  let rest = text;
  const found = [];
  for (const title of sorted) {
    const at = rest.indexOf(title);
    if (at === -1) {
      continue;
    }
    found.push(title);
    rest = `${rest.slice(0, at)}${rest.slice(at + title.length)}`;
  }
  return found;
}

function codexToolCalls(text) {
  const calls = [];
  for (const line of text.split("\n")) {
    if (!line.trim()) {
      continue;
    }
    let event;
    try {
      event = JSON.parse(line);
    } catch {
      continue;
    }
    if (event.type !== "item.completed" || event.item?.type !== "mcp_tool_call") {
      continue;
    }
    const item = event.item;
    const chunks = [];
    const content = item.result?.content;
    if (Array.isArray(content)) {
      for (const block of content) {
        if (typeof block?.text === "string") {
          chunks.push(block.text);
        }
      }
    }
    if (typeof item.error === "string") {
      chunks.push(item.error);
    } else if (item.error) {
      chunks.push(JSON.stringify(item.error));
    }
    calls.push({
      tool: item.tool || "",
      failed: item.status === "failed" || Boolean(item.error),
      code: item.arguments?.code || "",
      text: chunks.join("\n"),
    });
  }
  return calls;
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
