import { describe, expect, test } from "bun:test";
import { articleHost } from "./types";

describe("articleHost", () => {
  test("shows the site domain without www", () => {
    expect(articleHost("https://www.sellscale.com/blog-posts/our-agentic-engineering-org")).toBe(
      "sellscale.com",
    );
  });

  test("keeps a bare hostname", () => {
    expect(articleHost("https://example.com/post")).toBe("example.com");
  });

  test("returns the original text when it is not a url", () => {
    expect(articleHost("not a url")).toBe("not a url");
  });
});
