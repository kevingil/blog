import { defineConfig } from "@hey-api/openapi-ts";

export default defineConfig({
  input: "./openapi.json",
  output: "./src/client",
  plugins: [
    "@hey-api/typescript",
    {
      name: "@hey-api/sdk",
      operations: {
        strategy: "byTags",
      },
    },
  ],
});
