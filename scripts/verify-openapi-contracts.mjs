import { readFileSync } from "node:fs";

const root = new URL("../", import.meta.url);
const document = JSON.parse(readFileSync(new URL("openapi/openapi.json", root), "utf8"));

const methods = new Set(["get", "post", "put", "patch", "delete", "options", "head"]);
const operationIds = new Set();
let operations = 0;

for (const [path, pathItem] of Object.entries(document.paths)) {
  for (const [method, operation] of Object.entries(pathItem)) {
    if (!methods.has(method)) continue;
    operations += 1;
    const operationId = operation.operationId;
    if (typeof operationId !== "string" || operationId.length === 0) {
      throw new Error(`${method.toUpperCase()} ${path} has no stable operationId`);
    }
    if (operationIds.has(operationId)) {
      throw new Error(`duplicate operationId: ${operationId}`);
    }
    operationIds.add(operationId);
  }
}

console.log(`verified ${operations} OpenAPI operations and ${operationIds.size} unique operationIds`);
