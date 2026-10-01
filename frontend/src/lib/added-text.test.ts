import { describe, expect, test } from "bun:test";
import { addedTextRanges } from "./added-text";

describe("addedTextRanges", () => {
  test("highlights an inserted paragraph", () => {
    const before = "Intro\n\nEnding";
    const after = "Intro\n\nA new paragraph.\n\nEnding";
    expect(addedTextRanges(before, after)).toEqual([{ from: 7, to: 25 }]);
    expect(after.slice(7, 25)).toBe("A new paragraph.\n\n");
  });

  test("highlights a full replacement", () => {
    const ranges = addedTextRanges("old draft", "completely new");
    expect(ranges).toEqual([{ from: 0, to: "completely new".length }]);
  });

  test("leaves deletions unmarked", () => {
    expect(addedTextRanges("keep\nremove\nstay", "keep\nstay")).toEqual([]);
    expect(addedTextRanges("same", "same")).toEqual([]);
  });
});
