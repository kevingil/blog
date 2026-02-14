package tools

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// renderMarkdownToHTML converts markdown content to HTML using goldmark.
// Used by EditTextTool to persist the agent's markdown edits as HTML in draft_content.
func renderMarkdownToHTML(markdown string) string {
	var buf bytes.Buffer
	md := goldmark.New(
		goldmark.WithExtensions(extension.Table),
	)
	if err := md.Convert([]byte(markdown), &buf); err != nil {
		return ""
	}
	return buf.String()
}
