import { Extension, Mark, Node, mergeAttributes, type Editor } from '@tiptap/core';
import CodeBlock from '@tiptap/extension-code-block';
import { Plugin, PluginKey } from '@tiptap/pm/state';
import { Decoration, DecorationSet } from '@tiptap/pm/view';

const placeholderKey = new PluginKey('article-placeholder');

export const AddedHighlight = Mark.create({
  name: 'addedHighlight',
  inclusive: false,
  excludes: '',
  parseHTML() {
    return [{ tag: 'mark[data-added]' }];
  },
  renderHTML({ HTMLAttributes }) {
    return ['mark', mergeAttributes(HTMLAttributes, { 'data-added': 'true', class: 'added-edit' }), 0];
  },
});

export const ArticleLink = Mark.create({
  name: 'link',
  inclusive: false,
  excludes: 'link',
  addAttributes() {
    return {
      href: { default: null },
      title: { default: null },
    };
  },
  parseHTML() {
    return [{
      tag: 'a[href]',
      getAttrs: (element) => {
        const anchor = element as HTMLElement;
        const href = anchor.getAttribute('href');
        if (!href || href.toLowerCase().startsWith('javascript:')) return false;
        return { href, title: anchor.getAttribute('title') };
      },
    }];
  },
  renderHTML({ HTMLAttributes }) {
    return ['a', mergeAttributes(HTMLAttributes, { rel: 'noopener noreferrer', target: '_blank' }), 0];
  },
});

export const ArticleImage = Node.create({
  name: 'image',
  group: 'inline',
  inline: true,
  atom: true,
  draggable: true,
  addAttributes() {
    return {
      src: { default: null },
      alt: { default: null },
      title: { default: null },
      added: {
        default: null,
        parseHTML: (element) => (element.getAttribute('data-added') === 'true' ? true : null),
        renderHTML: (attributes) => (
          attributes.added ? { 'data-added': 'true', class: 'added-edit' } : {}
        ),
      },
    };
  },
  parseHTML() {
    return [{ tag: 'img[src]' }];
  },
  renderHTML({ HTMLAttributes }) {
    return ['img', mergeAttributes(HTMLAttributes)];
  },
});

export const ArticleCodeBlock = CodeBlock.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      added: {
        default: null,
        parseHTML: (element: HTMLElement) => (element.getAttribute('data-added') === 'true' ? true : null),
        renderHTML: (attributes: { added?: boolean | null }) => (
          attributes.added ? { 'data-added': 'true', class: 'added-edit' } : {}
        ),
      },
    };
  },
});

export const ArticlePlaceholder = Extension.create({
  name: 'articlePlaceholder',
  addProseMirrorPlugins() {
    return [
      new Plugin({
        key: placeholderKey,
        props: {
          decorations(state) {
            const { doc } = state;
            if (doc.childCount !== 1) return null;
            const first = doc.firstChild;
            if (!first?.isTextblock || first.content.size > 0) return null;
            return DecorationSet.create(doc, [
              Decoration.node(0, first.nodeSize, {
                class: 'is-editor-empty',
                'data-placeholder': 'Start writing…',
              }),
            ]);
          },
        },
      }),
    ];
  },
});

export function clearAddedHighlights(editor: Editor) {
  if (editor.isDestroyed) return;
  const { doc, schema } = editor.state;
  let transaction = editor.state.tr;
  let changed = false;
  const mark = schema.marks.addedHighlight;
  if (mark && doc.content.size > 0 && doc.rangeHasMark(0, doc.content.size, mark)) {
    transaction = transaction.removeMark(0, doc.content.size, mark);
    changed = true;
  }

  const updates: Array<{ pos: number; attrs: Record<string, unknown> }> = [];
  doc.descendants((node, pos) => {
    if ((node.type.name === 'codeBlock' || node.type.name === 'image') && node.attrs.added) {
      updates.push({ pos, attrs: { ...node.attrs, added: null } });
    }
  });
  for (let index = updates.length - 1; index >= 0; index -= 1) {
    const update = updates[index];
    transaction = transaction.setNodeMarkup(update.pos, undefined, update.attrs);
    changed = true;
  }
  if (!changed) return;
  editor.view.dispatch(transaction);
}
