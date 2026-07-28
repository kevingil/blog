import { readFileSync } from "node:fs";

const root = new URL("../", import.meta.url);
const document = JSON.parse(readFileSync(new URL("openapi/openapi.json", root), "utf8"));
const rows = readFileSync(new URL("docs/porting/CONTRACTS.tsv", root), "utf8")
  .trimEnd()
  .split("\n")
  .slice(1)
  .map((line) => line.split("\t"));

const methods = new Set(["get", "post", "put", "patch", "delete", "options", "head"]);
const normalize = (path) => path.replace(/\{[^}]+\}/g, "{}");
const operations = new Map();
const operationIds = new Set();

for (const [path, pathItem] of Object.entries(document.paths)) {
  for (const [method, operation] of Object.entries(pathItem)) {
    if (!methods.has(method)) continue;
    const operationId = operation.operationId;
    if (typeof operationId !== "string" || operationId.length === 0) {
      throw new Error(`${method.toUpperCase()} ${path} has no stable operationId`);
    }
    if (operationIds.has(operationId)) {
      throw new Error(`duplicate operationId: ${operationId}`);
    }
    operationIds.add(operationId);
    operations.set(`${method.toUpperCase()} ${normalize(path)}`, operationId);
  }
}

const missing = [];
for (const row of rows) {
  const [id, , method, path, , , , , , , classification] = row;
  if (
    !id.startsWith("HTTP-") ||
    classification !== "active-contract" ||
    path === "/swagger/*" ||
    path === "/api/openapi.json"
  ) {
    continue;
  }
  const key = `${method} ${normalize(path)}`;
  if (!operations.has(key)) missing.push(`${id} ${method} ${path}`);
}

if (missing.length > 0) {
  throw new Error(`OpenAPI is missing settled contracts:\n${missing.join("\n")}`);
}

console.log(
  `verified ${operations.size} OpenAPI operations, ${operationIds.size} unique operationIds, and all settled HTTP contract rows`,
);
