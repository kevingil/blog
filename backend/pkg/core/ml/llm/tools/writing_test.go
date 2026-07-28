package tools

import (
	"context"
	"encoding/json"
	"testing"
)

type recordingDraftSaver struct {
	articleID string
	content   string
}

func (s *recordingDraftSaver) UpdateDraftContent(_ context.Context, articleID string, htmlContent string) error {
	s.articleID = articleID
	s.content = htmlContent
	return nil
}

func TestReadDocumentAllowsEmptyDocument(t *testing.T) {
	ctx := WithDocumentContent(context.Background(), "", "")
	resp, err := NewReadDocumentTool().Run(ctx, ToolCall{})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if resp.IsError {
		t.Fatalf("expected empty document response to succeed, got error content %q", resp.Content)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		t.Fatalf("response content is not JSON: %v", err)
	}
	if result["total_lines"] != float64(0) {
		t.Fatalf("expected total_lines 0, got %#v", result["total_lines"])
	}
	if result["content"] != "" {
		t.Fatalf("expected empty content, got %#v", result["content"])
	}
}

func TestReplaceLinesCreatesInitialDraftForEmptyDocument(t *testing.T) {
	saver := &recordingDraftSaver{}
	ctx := WithArticleID(WithDocumentContent(context.Background(), "", ""), "article-1")
	resp, err := NewReplaceLinesTool(saver).Run(ctx, ToolCall{
		Input: `{"start_line":1,"end_line":1,"new_content":"Opening paragraph.\n\nSecond paragraph.","reason":"create draft"}`,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if resp.IsError {
		t.Fatalf("expected initial draft replacement to succeed, got error content %q", resp.Content)
	}

	want := "Opening paragraph.\n\nSecond paragraph."
	if got := GetDocumentMarkdownFromContext(ctx); got != want {
		t.Fatalf("expected context markdown %q, got %q", want, got)
	}
	if saver.articleID != "article-1" || saver.content != want {
		t.Fatalf("expected persisted draft (%q, %q), got (%q, %q)", "article-1", want, saver.articleID, saver.content)
	}
}
