export type TextRange = {
  from: number;
  to: number;
};

export function addedTextRanges(before: string, after: string): TextRange[] {
  if (before === after) return [];

  const beforeLines = before.split("\n");
  const afterLines = after.split("\n");
  let prefix = 0;
  const shared = Math.min(beforeLines.length, afterLines.length);
  while (prefix < shared && beforeLines[prefix] === afterLines[prefix]) {
    prefix += 1;
  }

  let suffix = 0;
  while (
    suffix < beforeLines.length - prefix &&
    suffix < afterLines.length - prefix &&
    beforeLines[beforeLines.length - 1 - suffix] === afterLines[afterLines.length - 1 - suffix]
  ) {
    suffix += 1;
  }

  const startLine = prefix;
  const endLine = afterLines.length - suffix;
  if (startLine >= endLine) return [];

  let from = 0;
  for (let index = 0; index < startLine; index += 1) {
    from += afterLines[index].length + 1;
  }

  let to = from;
  for (let index = startLine; index < endLine; index += 1) {
    to += afterLines[index].length;
    if (index < afterLines.length - 1) {
      to += 1;
    }
  }

  if (from >= to || to > after.length) return [];
  return [{ from, to }];
}
