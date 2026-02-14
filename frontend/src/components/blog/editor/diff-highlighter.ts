import { Extension } from '@tiptap/core';
import { Plugin, PluginKey } from 'prosemirror-state';
import { DOMParser as ProseDOMParser } from 'prosemirror-model';
import { Decoration, DecorationSet } from 'prosemirror-view';
import { diffWordsWithSpace } from 'diff';

// === TipTap Diff Extension ================================================
//
// Inline diff highlighting using word-level diffing.
// Key insight: both old and new content are parsed through ProseMirror's
// document model so text extraction is consistent on both sides.
// This avoids offset misalignment between DOM-based and PM-based text.
//
// =========================================================================

type DiffPart = { added?: boolean; removed?: boolean; value: string };

const DIFF_PLUGIN_KEY = new PluginKey('diff-highlighter');

// Extract plain text from a ProseMirror document, concatenating all text nodes.
function pmDocToText(doc: any): string {
  let text = '';
  doc.descendants((node: any) => {
    if (node.isText) {
      text += node.text || '';
    }
    return true;
  });
  return text;
}

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
          ({ tr, dispatch, editor }: { tr: unknown; dispatch: (tr: unknown) => void; editor: any }) => {
          try {
            const newDoc = (tr as any).doc;

            // Parse old HTML through the SAME ProseMirror schema so text
            // extraction is identical on both sides (no offset drift).
            const schema = editor.schema || newDoc.type.schema;
            const domNode = document.createElement('div');
            domNode.innerHTML = oldHtml;
            const oldDoc = ProseDOMParser.fromSchema(schema).parse(domNode);

            const oldText = pmDocToText(oldDoc);
            const newText = pmDocToText(newDoc);

            // Word-level diff that preserves whitespace tokens for accurate offsets
            const parts = diffWordsWithSpace(oldText, newText) as DiffPart[];

            // @ts-ignore
            this.storage.active = true;
            // @ts-ignore
            this.storage.parts = parts;
            // @ts-ignore
            (tr as any).setMeta(DIFF_PLUGIN_KEY, { updatedAt: Date.now(), active: true });
            if (dispatch) dispatch(tr);
            return true;
          } catch (e) {
            console.error('[DiffHighlighter] Failed to compute diff:', e);
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

            // Build text-node map for the NEW document (which is loaded in the editor)
            const textNodes: Array<{ from: number; to: number; text: string }> = [];
            doc.descendants((node: any, pos: number) => {
              if (node.isText) {
                textNodes.push({ from: pos, to: pos + node.nodeSize, text: node.text || '' });
              }
              return true;
            });

            // Map a plain-text offset to a ProseMirror position in the new document
            const offsetToPos = (offset: number): number => {
              let acc = 0;
              for (const n of textNodes) {
                const len = n.text.length;
                if (offset < acc + len) {
                  return n.from + (offset - acc);
                }
                acc += len;
              }
              // Clamp to end of last text node
              if (textNodes.length > 0) {
                return textNodes[textNodes.length - 1].to;
              }
              return Math.max(0, doc.content.size - 1);
            };

            let newIdx = 0;
            // @ts-ignore
            for (const part of ext.storage.parts) {
              if (part.added) {
                const from = offsetToPos(newIdx);
                const to = offsetToPos(newIdx + part.value.length);
                if (from < to && from >= 0) {
                  decorations.push(Decoration.inline(from, to, { class: 'diff-insert' }));
                }
                newIdx += part.value.length;
              } else if (part.removed) {
                const at = offsetToPos(newIdx);
                if (at >= 0) {
                  decorations.push(
                    Decoration.widget(
                      at,
                      () => {
                        const span = document.createElement('span');
                        span.className = 'diff-delete';
                        span.textContent = part.value;
                        return span;
                      },
                      { side: -1 }
                    )
                  );
                }
                // removed text does NOT advance newIdx
              } else {
                newIdx += part.value.length;
              }
            }

            try {
              return DecorationSet.create(doc, decorations);
            } catch (e) {
              console.error('[DiffHighlighter] Failed to create decorations:', e);
              return DecorationSet.empty;
            }
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
