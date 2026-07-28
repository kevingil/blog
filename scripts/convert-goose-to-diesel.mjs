import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const migrations = [
  "20250723064003_init",
  "20250813062742_update_project_table",
  "20260119004828_add_profile_and_org",
  "20260125045327_article_versions",
  "20260129000000_add_data_pipeline",
  "20260130000000_global_data_pipeline",
  "20260315000000_add_task_runs",
];

function body(sql, direction) {
  const marker = `-- +goose ${direction}\n-- +goose StatementBegin\n`;
  const start = sql.indexOf(marker);
  if (start < 0) {
    throw new Error(`missing ${direction} marker`);
  }
  const bodyStart = start + marker.length;
  const end = sql.indexOf("-- +goose StatementEnd", bodyStart);
  if (end < 0) {
    throw new Error(`missing ${direction} StatementEnd`);
  }
  return sql.slice(bodyStart, end);
}

for (const migration of migrations) {
  const source = join(
    root,
    "backend-go",
    "pkg",
    "database",
    "migrations",
    `${migration}.sql`,
  );
  const target = join(root, "backend", "migrations", migration);
  const sql = readFileSync(source, "utf8");
  mkdirSync(target, { recursive: true });
  writeFileSync(join(target, "up.sql"), body(sql, "Up"));
  writeFileSync(join(target, "down.sql"), body(sql, "Down"));
}
