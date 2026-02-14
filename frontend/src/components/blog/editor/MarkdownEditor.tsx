import CodeMirror from '@uiw/react-codemirror';
import { markdown } from '@codemirror/lang-markdown';
import { vscodeDark } from '@uiw/codemirror-theme-vscode';

interface MarkdownEditorProps {
  content: string;
  onChange: (value: string) => void;
  readOnly?: boolean;
}

export function MarkdownEditor({ content, onChange, readOnly }: MarkdownEditorProps) {
  return (
    <div className="h-full w-full overflow-hidden">
      <CodeMirror
        value={content}
        onChange={onChange}
        extensions={[markdown()]}
        theme={vscodeDark}
        readOnly={readOnly}
        basicSetup={{
          lineNumbers: true,
          foldGutter: true,
          highlightActiveLine: true,
          bracketMatching: true,
        }}
        className="text-sm"
        height="100%"
        width="100%"
      />
    </div>
  );
}
