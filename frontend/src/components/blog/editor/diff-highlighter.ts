import { Extension } from '@tiptap/core';
import { Plugin, PluginKey } from 'prosemirror-state';
import { Decoration, DecorationSet } from 'prosemirror-view';
import { diffWords } from 'diff';

// === TipTap Diff Extension ================================================
//
// This extension provides inline diff highlighting for the TipTap editor.
// It visualizes changes between old and new content with green (added) and
// red strikethrough (removed) decorations.
//
// ## Architecture
//
// The diff system operates in two modes:
//
// ### 1. PRECISE MODE (for edit_text operations)
// When we know exactly what was replaced and where:
// - Receives: originalText, newText, and the HTML index where the edit occurred
// - Extracts plain text from editor's document (NOT htmlToPlainText) for accuracy
// - Detects common edit patterns for reliable highlighting:
//   - INSERT BEFORE: Uses indexOf() to find preserved text position in editor
//   - INSERT AFTER: "Summary" → "Summary + New content" (new starts with original)
//   - PURE REPLACEMENT: Original text completely replaced
//   - COMPLEX: Falls back to word-level diff
// - Avoids false matches that occur when diffing entire documents
//
// ### 2. FULL DOCUMENT MODE (for rewrite_document operations)
// When the entire document is rewritten:
// - Compares full old vs new document plain text
// - Uses word-level diff (diffWords) for change detection
// - Includes structural change detection (same text, different HTML tag)
//
// ## Position Mapping
//
// HTML content is converted to plain text for diffing, then diff positions
// are mapped back to ProseMirror editor positions. Key considerations:
// - HTML tags don't appear in plain text but affect editor positions
// - Text node boundaries require careful offset-to-position mapping
// - The offsetToPos function handles gaps between text nodes
//
// CRITICAL: For PRECISE MODE, plain text MUST be extracted from the editor's
// document (tr.doc.descendants) rather than using htmlToPlainText(). This is
// because htmlToPlainText() can include whitespace (like newlines) that
// ProseMirror treats as structural boundaries, causing character count
// mismatches that result in incorrect highlight boundaries.
//
// ## Usage Flow
//
// 1. enterDiffPreview() is called with old/new HTML and optional editInfo
// 2. Editor content is set to new HTML
// 3. showDiff command computes diff parts based on mode
// 4. ProseMirror plugin applies decorations based on diff parts
// 5. User accepts (keeps new) or rejects (reverts to old)
//
// =========================================================================

// Represents a text segment with its HTML context and position
type TextSegment = {
  text: string;
  tagContext: string; // e.g., "h2", "p", "h3"
  startOffset: number; // position in plain text
  endOffset: number;
};

// Extract text segments with their tag context from HTML
function extractTextSegments(html: string): TextSegment[] {
  const div = document.createElement('div');
  div.innerHTML = html;
  const segments: TextSegment[] = [];
  let currentOffset = 0;
  
  function walk(node: Node, parentTag: string) {
    if (node.nodeType === Node.TEXT_NODE) {
      const text = node.textContent || '';
      if (text.length > 0) {
        segments.push({ 
          text, 
          tagContext: parentTag,
          startOffset: currentOffset,
          endOffset: currentOffset + text.length
        });
        currentOffset += text.length;
      }
    } else if (node.nodeType === Node.ELEMENT_NODE) {
      const el = node as Element;
      const tag = el.tagName.toLowerCase();
      for (const child of Array.from(el.childNodes)) {
        walk(child, tag);
      }
    }
  }
  
  for (const child of Array.from(div.childNodes)) {
    walk(child, 'root');
  }
  return segments;
}

// Extract plain text from HTML (for position mapping)
function htmlToPlainText(html: string): string {
  const div = document.createElement('div');
  div.innerHTML = html;
  return div.textContent || div.innerText || '';
}

// Type for diff parts (compatible with diffWords output from 'diff' library)
// - added: true if this text was inserted
// - removed: true if this text was deleted
// - value: the actual text content
// - count: optional word count (from diffWords)
type DiffPart = { added?: boolean; removed?: boolean; value: string; count?: number };

// Detect structural changes where text content is the same but HTML tag changed.
// For example, if "Summary" was in <h3> but is now in <h2>, this marks it as changed.
// Used in FULL DOCUMENT MODE to highlight tag-level modifications.
function detectStructuralChanges(
  oldSegments: TextSegment[],
  newSegments: TextSegment[],
  parts: DiffPart[]
): DiffPart[] {
  // Build a map of text -> tag for old content
  const oldTextToTag = new Map<string, string>();
  for (const seg of oldSegments) {
    const trimmed = seg.text.trim();
    if (trimmed) {
      oldTextToTag.set(trimmed, seg.tagContext);
    }
  }
  
  // Build a map of text -> tag for new content  
  const newTextToTag = new Map<string, string>();
  for (const seg of newSegments) {
    const trimmed = seg.text.trim();
    if (trimmed) {
      newTextToTag.set(trimmed, seg.tagContext);
    }
  }
  
  // Process parts and mark structural changes
  const result: DiffPart[] = [];
  
  for (const part of parts) {
    // Only check unchanged parts for structural changes
    if (!part.added && !part.removed) {
      const trimmed = part.value.trim();
      const oldTag = oldTextToTag.get(trimmed);
      const newTag = newTextToTag.get(trimmed);
      
      // If same text exists in both but with different tags, mark as changed
      if (oldTag && newTag && oldTag !== newTag) {
        result.push({ added: true, value: part.value });
      } else {
        result.push(part);
      }
    } else {
      result.push(part);
    }
  }
  
  return result;
}

const DIFF_PLUGIN_KEY = new PluginKey('diff-highlighter');
export const DiffHighlighter = Extension.create({
  name: 'diffHighlighter',
  addStorage() {
    return {
      active: false as boolean,
      parts: [] as DiffPart[],
    };
  },
  addCommands() {
    return {
      showDiff:
        (oldHtml: string, newHtml: string, editInfo?: { originalText: string; newText: string; htmlIndex: number }) =>
          ({ tr, dispatch }: { tr: unknown; dispatch: (tr: unknown) => void }) => {
          try {
            let parts: DiffPart[];
            
            if (editInfo) {
              // PRECISE MODE: Use exact edit boundaries for edit_text operations
              // This avoids false matches from diffing the entire document
              
              // Extract plain text from the original and new edit content
              const originalPlainText = htmlToPlainText(editInfo.originalText);
              const newPlainText = htmlToPlainText(editInfo.newText);
              
              // IMPORTANT: Get plain text from the EDITOR'S document, not htmlToPlainText
              // This ensures lengths match exactly with the text nodes used in offsetToPos
              // htmlToPlainText can include whitespace that the editor treats as structural
              const doc = (tr as any).doc;
              let fullNewPlainText = '';
              doc.descendants((node: any) => {
                if (node.isText) {
                  fullNewPlainText += node.text || '';
                }
                return true;
              });
              
              parts = [];
              
              // Find where the edit occurred by locating the HTML index position
              const beforeEditHtml = newHtml.substring(0, editInfo.htmlIndex);
              const editStartOffset = htmlToPlainText(beforeEditHtml).length;
              
              // Detect common edit patterns
              const isInsertBefore = newPlainText.endsWith(originalPlainText) && newPlainText.length > originalPlainText.length;
              const isInsertAfter = newPlainText.startsWith(originalPlainText) && newPlainText.length > originalPlainText.length;
              const isPureReplacement = !newPlainText.includes(originalPlainText);
              
              // Add unchanged prefix (content before the edit location)
              if (editStartOffset > 0) {
                parts.push({ value: fullNewPlainText.substring(0, editStartOffset) });
              }
              
              if (isInsertBefore) {
                // INSERT BEFORE: "Summary" → "New content...Summary"
                const preservedStartInEditor = fullNewPlainText.indexOf(originalPlainText, editStartOffset);
                
                if (preservedStartInEditor !== -1) {
                  const insertedValue = fullNewPlainText.substring(editStartOffset, preservedStartInEditor);
                  const preservedValue = fullNewPlainText.substring(preservedStartInEditor, preservedStartInEditor + originalPlainText.length);
                  
                  parts.push({ added: true, value: insertedValue });
                  parts.push({ value: preservedValue });
                  
                  const editEndOffset = preservedStartInEditor + originalPlainText.length;
                  if (editEndOffset < fullNewPlainText.length) {
                    parts.push({ value: fullNewPlainText.substring(editEndOffset) });
                  }
                } else {
                  const insertedLength = newPlainText.length - originalPlainText.length;
                  parts.push({ added: true, value: fullNewPlainText.substring(editStartOffset, editStartOffset + insertedLength) });
                  parts.push({ value: fullNewPlainText.substring(editStartOffset + insertedLength) });
                }
              } else if (isInsertAfter) {
                // INSERT AFTER: "Summary" → "Summary...New content"
                const preservedLength = originalPlainText.length;
                const insertedLength = newPlainText.length - originalPlainText.length;
                
                parts.push({ value: fullNewPlainText.substring(editStartOffset, editStartOffset + preservedLength) });
                parts.push({ added: true, value: fullNewPlainText.substring(editStartOffset + preservedLength, editStartOffset + preservedLength + insertedLength) });
                
                const editEndOffset = editStartOffset + newPlainText.length;
                if (editEndOffset < fullNewPlainText.length) {
                  parts.push({ value: fullNewPlainText.substring(editEndOffset) });
                }
              } else if (isPureReplacement) {
                // PURE REPLACEMENT: original completely replaced
                if (originalPlainText.length > 0) {
                  parts.push({ removed: true, value: originalPlainText });
                }
                parts.push({ added: true, value: fullNewPlainText.substring(editStartOffset, editStartOffset + newPlainText.length) });
                
                const editEndOffset = editStartOffset + newPlainText.length;
                if (editEndOffset < fullNewPlainText.length) {
                  parts.push({ value: fullNewPlainText.substring(editEndOffset) });
                }
              } else {
                // COMPLEX EDIT: Use word-level diff as fallback
                const editDiffParts = diffWords(originalPlainText, newPlainText);
                for (const part of editDiffParts) {
                  parts.push(part as DiffPart);
                }
                
                const editEndOffset = editStartOffset + newPlainText.length;
                if (editEndOffset < fullNewPlainText.length) {
                  parts.push({ value: fullNewPlainText.substring(editEndOffset) });
                }
              }
            } else {
              // FULL DOCUMENT MODE: For rewrite_document operations
              const oldText = htmlToPlainText(oldHtml);
              const newText = htmlToPlainText(newHtml);
              
              const rawParts = diffWords(oldText, newText);
              
              const oldSegments = extractTextSegments(oldHtml);
              const newSegments = extractTextSegments(newHtml);
              parts = detectStructuralChanges(oldSegments, newSegments, rawParts as DiffPart[]);
            }
            
            // @ts-ignore
            this.storage.active = true;
            // @ts-ignore
            this.storage.parts = parts;
            // @ts-ignore - set meta to force plugin to recompute decorations
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            (tr as any).setMeta(DIFF_PLUGIN_KEY, { updatedAt: Date.now(), active: true });
            if (dispatch) dispatch(tr);
            return true;
          } catch (e) {
            console.error('Failed to compute diff:', e);
            return false;
          }
        },
      clearDiff:
        () => ({ tr, dispatch }: { tr: unknown; dispatch: (tr: unknown) => void }) => {
          // @ts-ignore
          this.storage.active = false;
          // @ts-ignore
          this.storage.parts = [];
          // @ts-ignore
          (tr as any).setMeta(DIFF_PLUGIN_KEY, { updatedAt: Date.now(), active: false });
          if (dispatch) dispatch(tr);
          return true;
        },
    } as any;
  },
  addProseMirrorPlugins() {
    const ext = this;
    return [
      new Plugin({
        key: DIFF_PLUGIN_KEY,
        state: {
          init: () => DecorationSet.empty,
          apply(tr, _value) {
            // @ts-ignore
            if (!ext.storage.active || !(ext.storage.parts && ext.storage.parts.length)) {
              return DecorationSet.empty;
            }
            const doc = tr.doc;
            const decorations: Decoration[] = [];
            const textNodes: Array<{ from: number; to: number; text: string }> = [];
            doc.descendants((node, pos) => {
              if (node.isText) {
                textNodes.push({ from: pos, to: pos + node.nodeSize, text: node.text || '' });
              }
              return true;
            });

            const offsetToPos = (offset: number): number => {
              let acc = 0;
              for (const n of textNodes) {
                const len = n.text.length;
                if (offset < acc + len) {
                  const within = offset - acc;
                  return n.from + within;
                }
                acc += len;
              }
              if (textNodes.length > 0) {
                const last = textNodes[textNodes.length - 1];
                return last.to;
              }
              return doc.content.size - 1;
            };

            let newIdx = 0;
            // @ts-ignore
            for (const part of ext.storage.parts) {
              if (part.added) {
                const fromOff = newIdx;
                const toOff = newIdx + part.value.length;
                const from = offsetToPos(fromOff);
                const to = offsetToPos(toOff);
                if (from < to) {
                  decorations.push(Decoration.inline(from, to, { class: 'diff-insert' }));
                }
                newIdx += part.value.length;
              } else if (part.removed) {
                const at = offsetToPos(newIdx);
                const value = part.value;
                decorations.push(
                  Decoration.widget(
                    at,
                    () => {
                      const span = document.createElement('span');
                      span.className = 'diff-delete';
                      span.textContent = value;
                      return span;
                    },
                    { side: -1 }
                  )
                );
              } else {
                newIdx += part.value.length;
              }
            }

            const deco = DecorationSet.create(doc, decorations);
            return deco.map(tr.mapping, tr.doc);
          },
        },
        props: {
          decorations(state) {
            return (this as any).getState(state);
          },
        },
      }),
    ];
  },
});
