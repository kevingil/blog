import { useRef, useState, useEffect } from 'react';
import CodeMirror from '@uiw/react-codemirror';
import { markdown } from '@codemirror/lang-markdown';
import { vscodeDark } from '@uiw/codemirror-theme-vscode';
import { EditorView } from '@codemirror/view';

const lineWrapping = EditorView.lineWrapping;

interface MarkdownEditorProps {
  content: string;
  onChange: (value: string) => void;
  readOnly?: boolean;
}

export function MarkdownEditor({ content, onChange, readOnly }: MarkdownEditorProps) {
  const containerRef = useRef<HTMLDivElement>(null);
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

  return (
    <div ref={containerRef} style={{ width: '100%', height: '100%' }}>
      <CodeMirror
        value={content}
        onChange={onChange}
        extensions={[markdown(), lineWrapping]}
        theme={vscodeDark}
        readOnly={readOnly}
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
