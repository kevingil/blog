import { describe, expect, test } from "bun:test";
import { articleSchema } from "./editor-types";

describe("articleSchema", () => {
  test("accepts an empty draft", () => {
    const parsed = articleSchema.safeParse({
      title: "",
      content: "",
      image_url: "",
      tags: [],
      external_url: "",
    });
    expect(parsed.success).toBe(true);
  });

  test("rejects a title longer than 200 characters", () => {
    const parsed = articleSchema.safeParse({
      title: "t".repeat(201),
      content: "",
      tags: [],
    });
    expect(parsed.success).toBe(false);
  });
});
