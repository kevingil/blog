import { Extension } from '@tiptap/core';
import { Plugin, PluginKey } from 'prosemirror-state';
import { Decoration, DecorationSet } from 'prosemirror-view';

// === TipTap Diff Extension ================================================
//
// Provides inline diff highlighting for the TipTap editor.
// Shows changes between old and new content with green (added) and
// red strikethrough (removed) decorations.
//
// ## How it works
//
// 1. enterDiffPreview() sets editor content to newHtml and calls showDiff(oldHtml, newHtml)
// 2. showDiff extracts plain text from both the old HTML (via DOM) and the new editor document
// 3. Character-by-character comparison finds the exact divergence boundaries
// 4. Produces exactly 2-4 parts: prefix + removed (red) + added (green) + suffix
// 5. ProseMirror plugin applies decorations based on these parts
// 6. User accepts (keeps new) or rejects (reverts to old)
//
// This approach is deterministic and handles repeated words, single-character
// changes, and all edit types without ambiguity.
//
// =========================================================================

// Extract plain text from HTML the same way ProseMirror does:
// concatenate all text node content without separators between blocks.
// This ensures offsets match the editor's document text exactly.
function extractDocText(html: string): string {
  const div = document.createElement('div');
  div.innerHTML = html;
  let text = '';
  function walk(node: Node) {
    if (node.nodeType === Node.TEXT_NODE) {
      text += node.textContent || '';
    } else {
      for (const child of Array.from(node.childNodes)) {
        walk(child);
      }
    }
  }
  walk(div);
  return text;
}

// Type for diff parts
// - added: true if this text was inserted (green highlight)
// - removed: true if this text was deleted (red strikethrough)
// - value: the actual text content
type DiffPart = { added?: boolean; removed?: boolean; value: string };

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
        (oldHtml: string, _newHtml: string) =>
          ({ tr, dispatch }: { tr: unknown; dispatch: (tr: unknown) => void }) => {
          try {
            // Extract plain text from old HTML (same way ProseMirror extracts from its document)
            const oldDocText = extractDocText(oldHtml);

            // Extract plain text from the editor's current document (which has the new content)
            const doc = (tr as any).doc;
            let newDocText = '';
            doc.descendants((node: any) => {
              if (node.isText) {
                newDocText += node.text || '';
              }
              return true;
            });

            // Character-by-character comparison: find where old and new diverge from the start
            let start = 0;
            while (start < oldDocText.length && start < newDocText.length
                   && oldDocText[start] === newDocText[start]) {
              start++;
            }

            // Find where they diverge from the end
            let oldEnd = oldDocText.length;
            let newEnd = newDocText.length;
            while (oldEnd > start && newEnd > start
                   && oldDocText[oldEnd - 1] === newDocText[newEnd - 1]) {
              oldEnd--;
              newEnd--;
            }

            // Build exactly 2-4 clean parts: prefix + removed + added + suffix
            const parts: DiffPart[] = [];
            if (start > 0) {
              parts.push({ value: newDocText.substring(0, start) });
            }
            if (oldEnd > start) {
              parts.push({ removed: true, value: oldDocText.substring(start, oldEnd) });
            }
            if (newEnd > start) {
              parts.push({ added: true, value: newDocText.substring(start, newEnd) });
            }
            if (newEnd < newDocText.length) {
              parts.push({ value: newDocText.substring(newEnd) });
            }

            // @ts-ignore
            this.storage.active = true;
            // @ts-ignore
            this.storage.parts = parts;
            // @ts-ignore
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

            // Map a plain text offset to a ProseMirror document position
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
