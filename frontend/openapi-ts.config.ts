import { defineConfig } from "@hey-api/openapi-ts";

export default defineConfig({
  input: "../backend/docs/swagger.json",
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
