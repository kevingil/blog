import { useRef, useState, useEffect, type MutableRefObject } from 'react';
import CodeMirror from '@uiw/react-codemirror';
import { markdown } from '@codemirror/lang-markdown';
import { vscodeDark } from '@uiw/codemirror-theme-vscode';
import { EditorView, Decoration, type DecorationSet } from '@codemirror/view';
import { StateEffect, StateField } from '@codemirror/state';
import type { TextRange } from '@/lib/added-text';

const lineWrapping = EditorView.lineWrapping;

const setAddedHighlights = StateEffect.define<TextRange[]>();

const addedHighlightField = StateField.define<DecorationSet>({
  create() {
    return Decoration.none;
  },
  update(decorations, transaction) {
    for (const effect of transaction.effects) {
      if (effect.is(setAddedHighlights)) {
        const docLength = transaction.state.doc.length;
        const marks = effect.value
          .filter((range) => range.from >= 0 && range.to <= docLength && range.from < range.to)
          .map((range) => Decoration.mark({ class: 'cm-added-text' }).range(range.from, range.to));
        return Decoration.set(marks, true);
      }
    }
    if (transaction.docChanged) return Decoration.none;
    return decorations.map(transaction.changes);
  },
  provide: (field) => EditorView.decorations.from(field),
});

const addedHighlightTheme = EditorView.baseTheme({
  '.cm-added-text': {
    backgroundColor: 'rgba(74, 222, 128, 0.28)',
    borderRadius: '2px',
  },
});

interface MarkdownEditorProps {
  content: string;
  onChange: (value: string) => void;
  readOnly?: boolean;
  editorViewRef?: MutableRefObject<EditorView | null>;
  highlights?: TextRange[];
}

export function MarkdownEditor({
  content,
  onChange,
  readOnly,
  editorViewRef,
  highlights = [],
}: MarkdownEditorProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const localViewRef = useRef<EditorView | null>(null);
  const [height, setHeight] = useState(400);

  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    const observer = new ResizeObserver((entries) => {
      for (const entry of entries) {
        setHeight(entry.contentRect.height);
      }
    });
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  useEffect(() => {
    const view = localViewRef.current;
    if (!view) return;
    view.dispatch({ effects: setAddedHighlights.of(highlights) });
  }, [highlights, content]);

  return (
    <div ref={containerRef} style={{ width: '100%', height: '100%' }}>
      <CodeMirror
        value={content}
        onChange={(value, viewUpdate) => {
          const fromUser = viewUpdate.transactions.some(
            (transaction) =>
              transaction.isUserEvent('input') ||
              transaction.isUserEvent('delete') ||
              transaction.isUserEvent('undo') ||
              transaction.isUserEvent('redo'),
          );
          if (fromUser) onChange(value);
        }}
        extensions={[markdown(), lineWrapping, addedHighlightField, addedHighlightTheme]}
        theme={vscodeDark}
        readOnly={readOnly}
        onCreateEditor={(view) => {
          localViewRef.current = view;
          if (editorViewRef) editorViewRef.current = view;
        }}
        basicSetup={{
          lineNumbers: true,
          foldGutter: true,
          highlightActiveLine: true,
          bracketMatching: true,
        }}
        className="text-sm"
        height={`${height}px`}
      />
    </div>
  );
}
