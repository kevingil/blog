import { readdirSync, statSync, writeFileSync } from "node:fs";
import { dirname, join, relative, sep } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const goRoot = join(root, "backend-go");

function walk(path) {
  return readdirSync(path)
    .sort()
    .flatMap((name) => {
      const child = join(path, name);
      return statSync(child).isDirectory() ? walk(child) : [child];
    });
}

function destination(source) {
  const local = source.slice("backend-go/".length);
  if (local === "main.go") return "backend/src/main.rs";
  if (local.startsWith("cmd/")) {
    return `backend/src/bin/${local.slice(4, -3)}.rs`;
  }
  if (local.startsWith("docs/")) return "backend/src/openapi.rs";
  if (local.startsWith("testutil/mocks/")) {
    return `backend/tests/support/mocks/${local.slice("testutil/mocks/".length, -3)}.rs`;
  }
  if (local === "pkg/api/router.go") return "backend/src/app.rs";
  if (local === "pkg/core/errors.go") return "backend/src/error.rs";
  if (local === "pkg/database/database.go") {
    return "backend/src/database/pool.rs;backend/src/bootstrap.rs";
  }
  if (local.startsWith("pkg/")) {
    return `backend/src/${local.slice(4, -3)}.rs`;
  }
  throw new Error(`no explicit destination rule for ${source}`);
}

function phase(source) {
  if (source.includes("/pkg/types/") || source.includes("/pkg/config/") || source.includes("/pkg/constants/") || source.endsWith("/pkg/core/errors.go")) return "4.1";
  if (source.includes("/pkg/database/") || source.includes("/cmd/")) return "4.2";
  if (source.includes("/pkg/core/worker/")) return "4.6";
  if (source.includes("/pkg/core/copilot/") || source.includes("/pkg/core/ml/")) return "4.5";
  if (source.includes("/pkg/core/")) return "4.3";
  if (source.includes("/pkg/integrations/")) return "4.4";
  if (source.includes("/pkg/api/")) return "4.7";
  return "4.8";
}

const rows = walk(goRoot)
  .filter((path) => path.endsWith(".go"))
  .map((path) => relative(root, path).split(sep).join("/"))
  .map((source) => [source, destination(source), phase(source), "pending", ""])
  .sort((left, right) => left[0].localeCompare(right[0]));

const header = ["go_source", "rust_destination", "phase", "status", "approved_disposition"];
writeFileSync(
  join(root, "docs", "porting", "MODULE_MAP.tsv"),
  [header, ...rows].map((row) => row.join("\t")).join("\n") + "\n",
);
