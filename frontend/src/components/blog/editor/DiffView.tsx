import ReactDiffViewer, { DiffMethod } from 'react-diff-viewer-continued';
import { Button } from '@/components/ui/button';
import { Check, X } from 'lucide-react';

interface DiffViewProps {
  oldValue: string;
  newValue: string;
  onAccept: () => void;
  onReject: () => void;
}

export function DiffView({ oldValue, newValue, onAccept, onReject }: DiffViewProps) {
  const hasChanges = oldValue !== newValue;

  return (
    <div className="flex flex-col h-full">
      {hasChanges ? (
        <>
          <div className="flex-1 overflow-auto">
            <ReactDiffViewer
              oldValue={oldValue}
              newValue={newValue}
              splitView={true}
              useDarkTheme={true}
              compareMethod={DiffMethod.WORDS}
              leftTitle="Before"
              rightTitle="After"
              styles={{
                contentText: { fontSize: '13px', lineHeight: '1.5' },
              }}
            />
          </div>
          <div className="flex items-center justify-end gap-2 p-3 border-t border-border bg-background/80">
            <Button variant="outline" size="sm" onClick={onReject}>
              <X className="h-4 w-4 mr-1" />
              Reject
            </Button>
            <Button size="sm" onClick={onAccept}>
              <Check className="h-4 w-4 mr-1" />
              Keep All
            </Button>
          </div>
        </>
      ) : (
        <div className="flex-1 flex items-center justify-center text-muted-foreground">
          No changes to review
        </div>
      )}
    </div>
  );
}
