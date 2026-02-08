package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"backend/pkg/database/models"

	"github.com/google/uuid"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// ArticleSourceService interface for source operations - using the real service directly
type ArticleSourceService interface {
	SearchSimilarSources(ctx context.Context, articleID uuid.UUID, query string, limit int) ([]*models.Source, error)
}

// TextGenerationService interface for text generation operations
type TextGenerationService interface {
	GenerateImagePrompt(ctx context.Context, content string) (string, error)
}

// ReadDocumentTool allows the agent to read the current document content with line numbers
type ReadDocumentTool struct{}

func NewReadDocumentTool() *ReadDocumentTool {
	return &ReadDocumentTool{}
}

func (t *ReadDocumentTool) Info() ToolInfo {
	return ToolInfo{
		Name: "read_document",
		Description: `Read the current document content (Markdown format) with line numbers.

WHEN TO USE:
- Before making any edits with edit_text
- When you need to see the full content
- When you need to find specific text to edit

OUTPUT FORMAT:
Lines are numbered for easy reference. Content is Markdown:
   1| ## Introduction
   2| This is the first paragraph...
   3|
   4| ### Code Example
   5| ` + "```" + `go
   6| func main() {}
   7| ` + "```" + `

Use line numbers to reference specific locations when discussing edits.
Include enough surrounding context in old_str to ensure uniqueness.`,
		Parameters: map[string]any{
			"start_line": map[string]any{
				"type":        []string{"number", "null"},
				"description": "Optional: Start reading from this line number (1-indexed)",
			},
			"end_line": map[string]any{
				"type":        []string{"number", "null"},
				"description": "Optional: Stop reading at this line number (inclusive)",
			},
		},
		Required: []string{},
	}
}

func (t *ReadDocumentTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
	var input struct {
		StartLine int `json:"start_line"`
		EndLine   int `json:"end_line"`
	}
	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		// Ignore unmarshal errors for optional params
		log.Printf("📖 [ReadDocument] No line range specified, reading full document")
	}

	// Get markdown content (preferred) or fall back to HTML
	docContent := GetDocumentMarkdownFromContext(ctx)
	if docContent == "" {
		docContent = GetDocumentHTMLFromContext(ctx)
	}
	if docContent == "" {
		log.Printf("📖 [ReadDocument] ERROR: No document content in context")
		return NewTextErrorResponse("No document content available. The document may be empty or not loaded."), nil
	}

	lines := strings.Split(docContent, "\n")
	totalLines := len(lines)

	// Apply line range if specified (1-indexed)
	start := 0
	end := totalLines
	if input.StartLine > 0 {
		start = input.StartLine - 1
		if start >= totalLines {
			return NewTextErrorResponse(fmt.Sprintf("Start line %d exceeds document length (%d lines)", input.StartLine, totalLines)), nil
		}
	}
	if input.EndLine > 0 {
		end = input.EndLine
		if end > totalLines {
			end = totalLines
		}
	}

	// Format with line numbers
	var numbered []string
	for i := start; i < end; i++ {
		numbered = append(numbered, fmt.Sprintf("%4d| %s", i+1, lines[i]))
	}

	content := strings.Join(numbered, "\n")

	log.Printf("📖 [ReadDocument] Returning lines %d-%d of %d total lines", start+1, end, totalLines)

	result := map[string]interface{}{
		"content":     content,
		"total_lines": totalLines,
		"showing":     fmt.Sprintf("lines %d-%d of %d", start+1, end, totalLines),
		"tool_name":   "read_document",
	}

	resultJSON, _ := json.Marshal(result)
	return NewTextResponse(string(resultJSON)), nil
}

// GetRelevantSourcesTool finds relevant source chunks based on query
type GetRelevantSourcesTool struct {
	sourceService ArticleSourceService
}

func NewGetRelevantSourcesTool(sourceService ArticleSourceService) *GetRelevantSourcesTool {
	return &GetRelevantSourcesTool{
		sourceService: sourceService,
	}
}

func (t *GetRelevantSourcesTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "get_relevant_sources",
		Description: "Find relevant source chunks based on a query to provide context for document rewriting",
		Parameters: map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "The query to search for relevant sources (e.g., main topics, keywords from the document)",
			},
			"limit": map[string]any{
				"type":        []string{"number", "null"},
				"description": "Maximum number of relevant sources to return (default: 5)",
			},
		},
		Required: []string{"query"},
	}
}

func (t *GetRelevantSourcesTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
	var input struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}

	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		log.Printf("🔍 [GetRelevantSources] ERROR: Failed to parse input: %v", err)
		return NewTextErrorResponse("Invalid input format"), err
	}

	if input.Query == "" {
		log.Printf("🔍 [GetRelevantSources] ERROR: Empty query provided")
		return NewTextErrorResponse("query is required"), fmt.Errorf("query is required")
	}

	// Set default limit
	if input.Limit <= 0 {
		input.Limit = 5
	}

	log.Printf("🔍 [GetRelevantSources] Starting source search")
	log.Printf("   📝 Query: %q", input.Query)
	log.Printf("   🎯 Limit: %d", input.Limit)

	// Debug context values
	sessionID, messageID := GetContextValues(ctx)
	log.Printf("   🔍 Context Debug - Session ID: %q", sessionID)
	log.Printf("   🔍 Context Debug - Message ID: %q", messageID)

	// Get article ID from context
	articleIDStr := GetArticleIDFromContext(ctx)
	log.Printf("   🔍 Context Debug - Article ID: %q", articleIDStr)

	if articleIDStr == "" {
		log.Printf("🔍 [GetRelevantSources] WARNING: No article ID in context - cannot search for article-specific sources")

		// Return empty result instead of error to allow the rewrite to continue without sources
		result := map[string]interface{}{
			"relevant_sources": []map[string]interface{}{},
			"query":            input.Query,
			"total_found":      0,
			"tool_name":        "get_relevant_sources",
			"warning":          "No article ID available - returned empty sources",
		}

		log.Printf("🔍 [GetRelevantSources] ⚠️  Returning empty sources due to missing article ID")
		resultJSON, _ := json.Marshal(result)
		return NewTextResponse(string(resultJSON)), nil
	}

	articleID, err := uuid.Parse(articleIDStr)
	if err != nil {
		log.Printf("🔍 [GetRelevantSources] ERROR: Invalid article ID format: %s", articleIDStr)
		return NewTextErrorResponse("Invalid article ID"), fmt.Errorf("invalid article ID: %w", err)
	}

	log.Printf("   📄 Article ID: %s", articleID)

	// Search for similar sources
	log.Printf("🔍 [GetRelevantSources] Executing vector similarity search...")
	sources, err := t.sourceService.SearchSimilarSources(ctx, articleID, input.Query, input.Limit)
	if err != nil {
		log.Printf("🔍 [GetRelevantSources] ERROR: Search failed: %v", err)
		return NewTextErrorResponse(fmt.Sprintf("Failed to search sources: %v", err)), err
	}

	log.Printf("🔍 [GetRelevantSources] ✅ Found %d sources", len(sources))

	// Convert models.Source to response format with text chunking
	var relevantSources []map[string]interface{}
	for i, source := range sources {
		// Log detailed information about each source
		contentLength := len(source.Content)
		contentPreview := source.Content
		if len(contentPreview) > 150 {
			contentPreview = contentPreview[:150] + "..."
		}

		log.Printf("🔍 [GetRelevantSources] Source #%d:", i+1)
		log.Printf("   📋 Title: %q", source.Title)
		log.Printf("   🔗 URL: %q", source.URL)
		log.Printf("   📊 Type: %q", source.SourceType)
		log.Printf("   📏 Content Length: %d characters", contentLength)
		log.Printf("   📝 Content Preview: %q", contentPreview)

		// Chunk the content and find the most relevant chunks
		chunks := t.chunkText(source.Content, 1200)                        // 1200 character chunks with overlap for more context
		relevantChunks := t.findMostRelevantChunks(chunks, input.Query, 2) // Top 2 chunks per source (longer chunks)

		log.Printf("   🧩 Generated %d chunks, selected %d most relevant", len(chunks), len(relevantChunks))

		// Add each relevant chunk as a separate source entry
		for j, chunk := range relevantChunks {
			chunkPreview := chunk.Text
			if len(chunkPreview) > 200 {
				chunkPreview = chunkPreview[:200] + "..."
			}

			log.Printf("   📝 Chunk #%d (score: %.3f, length: %d chars): %q", j+1, chunk.Score, len(chunk.Text), chunkPreview)

			sourceData := map[string]interface{}{
				"source_title": source.Title,
				"source_url":   source.URL,
				"text_chunk":   chunk.Text,
				"source_type":  source.SourceType,
				"chunk_score":  chunk.Score,
				"chunk_index":  j + 1,
			}
			relevantSources = append(relevantSources, sourceData)
		}
	}

	// Calculate and log some quality metrics
	totalContentLength := 0
	totalChunks := 0
	for _, source := range sources {
		totalContentLength += len(source.Content)
	}
	totalChunks = len(relevantSources) // Now each chunk is a separate entry

	// Calculate chunk size statistics
	var chunkSizes []int
	totalChunkLength := 0
	for _, source := range relevantSources {
		if chunk, ok := source["text_chunk"].(string); ok {
			chunkLength := len(chunk)
			chunkSizes = append(chunkSizes, chunkLength)
			totalChunkLength += chunkLength
		}
	}

	log.Printf("🔍 [GetRelevantSources] 📊 Quality Metrics:")
	log.Printf("   📄 Total sources found: %d", len(sources))
	log.Printf("   🧩 Total chunks extracted: %d", totalChunks)
	log.Printf("   📏 Total original content length: %d characters", totalContentLength)
	log.Printf("   📏 Total chunk content length: %d characters", totalChunkLength)
	if len(sources) > 0 {
		avgContentLength := totalContentLength / len(sources)
		avgChunksPerSource := float64(totalChunks) / float64(len(sources))
		log.Printf("   📊 Average content length per source: %d characters", avgContentLength)
		log.Printf("   📊 Average chunks per source: %.1f", avgChunksPerSource)
	}
	if len(chunkSizes) > 0 {
		avgChunkSize := totalChunkLength / len(chunkSizes)
		minChunkSize := chunkSizes[0]
		maxChunkSize := chunkSizes[0]
		for _, size := range chunkSizes {
			if size < minChunkSize {
				minChunkSize = size
			}
			if size > maxChunkSize {
				maxChunkSize = size
			}
		}
		log.Printf("   📊 Chunk sizes - Avg: %d, Min: %d, Max: %d characters", avgChunkSize, minChunkSize, maxChunkSize)
	}

	result := map[string]interface{}{
		"relevant_sources": relevantSources,
		"query":            input.Query,
		"total_found":      len(relevantSources),
		"tool_name":        "get_relevant_sources",
	}

	log.Printf("🔍 [GetRelevantSources] ✅ Returning %d relevant chunks from %d sources", len(relevantSources), len(sources))

	// Create artifact hint for sources display
	artifactData := map[string]interface{}{
		"sources":     relevantSources,
		"query":       input.Query,
		"total_found": len(relevantSources),
	}

	resultJSON, _ := json.Marshal(result)
	return ToolResponse{
		Type:    ToolResponseTypeText,
		Content: string(resultJSON),
		Result:  result,
		Artifact: &ArtifactHint{
			Type: ArtifactHintTypeSources,
			Data: artifactData,
		},
	}, nil
}

// TextChunk represents a chunk of text with relevance scoring
type TextChunk struct {
	Text  string
	Score float64
	Index int
}

// chunkText splits text into overlapping chunks for better context preservation
func (t *GetRelevantSourcesTool) chunkText(text string, chunkSize int) []TextChunk {
	if len(text) <= chunkSize {
		return []TextChunk{{Text: text, Index: 0}}
	}

	var chunks []TextChunk
	overlap := chunkSize / 3 // 33% overlap for better context preservation

	for i := 0; i < len(text); i += chunkSize - overlap {
		end := i + chunkSize
		if end > len(text) {
			end = len(text)
		}

		chunk := text[i:end]
		// Try to break at sentence boundaries to avoid cutting words
		if end < len(text) {
			// Look for the last sentence boundary in the last third of the chunk
			searchStart := len(chunk) * 2 / 3
			if searchStart < len(chunk) {
				lastPart := chunk[searchStart:]
				if lastDot := strings.LastIndex(lastPart, "."); lastDot != -1 {
					chunk = chunk[:searchStart+lastDot+1]
				} else if lastQuestion := strings.LastIndex(lastPart, "?"); lastQuestion != -1 {
					chunk = chunk[:searchStart+lastQuestion+1]
				} else if lastExclamation := strings.LastIndex(lastPart, "!"); lastExclamation != -1 {
					chunk = chunk[:searchStart+lastExclamation+1]
				}
			}
		}

		chunks = append(chunks, TextChunk{
			Text:  strings.TrimSpace(chunk),
			Index: len(chunks),
		})

		if end >= len(text) {
			break
		}
	}

	return chunks
}

// findMostRelevantChunks finds the most relevant chunks using simple text similarity
func (t *GetRelevantSourcesTool) findMostRelevantChunks(chunks []TextChunk, query string, maxChunks int) []TextChunk {
	if len(chunks) == 0 {
		return chunks
	}

	// Score each chunk based on keyword overlap with query
	queryWords := t.extractKeywords(strings.ToLower(query))

	for i := range chunks {
		chunks[i].Score = t.calculateRelevanceScore(chunks[i].Text, queryWords)
	}

	// Sort by score (highest first)
	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].Score > chunks[j].Score
	})

	// Return top chunks, but limit to maxChunks
	if len(chunks) > maxChunks {
		chunks = chunks[:maxChunks]
	}

	return chunks
}

// extractKeywords extracts meaningful keywords from a query
func (t *GetRelevantSourcesTool) extractKeywords(text string) []string {
	// Simple keyword extraction - split on spaces and filter common words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
		"be": true, "been": true, "being": true, "have": true, "has": true, "had": true,
		"do": true, "does": true, "did": true, "will": true, "would": true, "could": true,
		"should": true, "may": true, "might": true, "must": true, "can": true,
		"this": true, "that": true, "these": true, "those": true, "i": true, "you": true,
		"he": true, "she": true, "it": true, "we": true, "they": true, "me": true,
		"him": true, "her": true, "us": true, "them": true,
	}

	words := strings.Fields(text)
	var keywords []string

	for _, word := range words {
		// Clean the word
		word = strings.ToLower(strings.Trim(word, ".,!?;:()[]{}\"'"))

		// Skip if empty, too short, or a stop word
		if len(word) < 3 || stopWords[word] {
			continue
		}

		keywords = append(keywords, word)
	}

	return keywords
}

// calculateRelevanceScore calculates a simple relevance score based on keyword frequency
func (t *GetRelevantSourcesTool) calculateRelevanceScore(text string, queryKeywords []string) float64 {
	if len(queryKeywords) == 0 {
		return 0.0
	}

	textLower := strings.ToLower(text)
	textWords := strings.Fields(textLower)
	textWordCount := make(map[string]int)

	// Count word frequencies in text
	for _, word := range textWords {
		word = strings.Trim(word, ".,!?;:()[]{}\"'")
		if len(word) > 2 {
			textWordCount[word]++
		}
	}

	// Calculate score based on keyword matches
	var score float64
	matchedKeywords := 0

	for _, keyword := range queryKeywords {
		if count, exists := textWordCount[keyword]; exists {
			// Use TF-IDF inspired scoring: frequency * log(text_length / keyword_frequency)
			tf := float64(count) / float64(len(textWords))
			idf := math.Log(float64(len(textWords)) / float64(count))
			score += tf * idf
			matchedKeywords++
		}
	}

	// Boost score based on percentage of matched keywords
	keywordCoverage := float64(matchedKeywords) / float64(len(queryKeywords))
	score *= (1.0 + keywordCoverage)

	return score
}

// EditTextTool edits specific text in the document
type EditTextTool struct{}

func NewEditTextTool() *EditTextTool {
	return &EditTextTool{}
}

func (t *EditTextTool) Info() ToolInfo {
	return ToolInfo{
		Name: "edit_text",
		Description: `Edit specific text in the document using exact string replacement. Returns a diff for user approval.

BEFORE USING: Call read_document first to see the content with line numbers.

The document content is in Markdown format. You find text (old_str) and replace it with new text (new_str).

CRITICAL REQUIREMENTS:
1. old_str must EXACTLY match text in the document (character-for-character, including whitespace and newlines)
2. old_str must be UNIQUE in the document - include enough surrounding context
3. Keep edits focused - one logical change at a time
4. Content is in Markdown format - use markdown syntax for headings, code blocks, lists, etc.

EXAMPLES:

BAD (not unique):
  old_str: "Introduction"

GOOD (unique with context):
  old_str: "## Introduction\n\nOracle announced JavaScript support"

BAD (too large - causes JSON errors):
  old_str: [entire 500-word section]

GOOD (focused edit):
  old_str: "Teams already writing business logic in JavaScript can move that code"
  new_str: "Teams with existing JavaScript expertise can migrate business logic"

GOOD (editing a code block):
  old_str: "` + "```" + `go\nfunc main() {\n    fmt.Println(\"hello\")\n}\n` + "```" + `"
  new_str: "` + "```" + `go\nfunc main() {\n    fmt.Println(\"hello world\")\n}\n` + "```" + `"

NEVER include a title/heading at the start of new_str - titles are managed separately.
Write like a human - avoid puffery, hedging, and AI patterns.`,
		Parameters: map[string]any{
			"old_str": map[string]any{
				"type":        "string",
				"description": "The exact markdown text to find and replace (must be unique, include surrounding context)",
			},
			"new_str": map[string]any{
				"type":        "string",
				"description": "The replacement markdown text. No title/heading at start.",
			},
			"reason": map[string]any{
				"type":        "string",
				"description": "Brief explanation of the edit",
			},
		},
		Required: []string{"old_str", "new_str", "reason"},
	}
}

func (t *EditTextTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
	var input struct {
		OldStr string `json:"old_str"`
		NewStr string `json:"new_str"`
		Reason string `json:"reason"`
	}

	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return NewTextErrorResponse("Invalid input format"), err
	}

	if input.OldStr == "" || input.NewStr == "" {
		return NewTextErrorResponse("old_str and new_str are required"), fmt.Errorf("old_str and new_str are required")
	}

	log.Printf("✏️ [EditText] Processing markdown edit")
	log.Printf("   📝 Reason: %q", input.Reason)
	log.Printf("   📄 old_str length: %d chars", len(input.OldStr))
	log.Printf("   📄 new_str length: %d chars", len(input.NewStr))

	// Get the document markdown from context to validate the edit
	documentMarkdown := GetDocumentMarkdownFromContext(ctx)

	// #region agent log
	// Log: what does the old_str look like byte-by-byte for first 80 chars, and does the doc contain the first line?
	oldStrFirst80 := input.OldStr; if len(oldStrFirst80) > 80 { oldStrFirst80 = oldStrFirst80[:80] }
	firstNewline := strings.Index(input.OldStr, "\n")
	firstLine := input.OldStr; if firstNewline > 0 { firstLine = input.OldStr[:firstNewline] }
	firstLineInDoc := strings.Contains(documentMarkdown, firstLine)
	hasFenced := strings.Contains(documentMarkdown, "```")
	// Find closest match by searching for progressively shorter prefixes
	matchLen := 0
	for i := len(firstLine); i > 10; i-- {
		if strings.Contains(documentMarkdown, firstLine[:i]) { matchLen = i; break }
	}
	debugEntry := fmt.Sprintf(`{"location":"writing.go:edit_text","message":"edit validation detail","data":{"oldStrLen":%d,"docLen":%d,"firstLine":%q,"firstLineInDoc":%v,"firstLineLen":%d,"matchLen":%d,"hasFenced":%v,"oldStrBytes":%q},"timestamp":%d,"hypothesisId":"H1,H3"}`,
		len(input.OldStr), len(documentMarkdown), firstLine, firstLineInDoc, len(firstLine), matchLen, hasFenced,
		[]byte(oldStrFirst80), time.Now().UnixMilli())
	if f, err := os.OpenFile("/Users/kgil/Git/blogs/blog-agent-go/.cursor/debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil { f.WriteString(debugEntry + "\n"); f.Close() }
	// #endregion

	var newMarkdown string
	if documentMarkdown != "" {
		// Validate: old_str must exist in the document
		index := strings.Index(documentMarkdown, input.OldStr)

		// If exact match fails, try normalizing markdown escapes, JSON unicode escapes, and whitespace
		if index == -1 {
			normalizer := strings.NewReplacer(
				// Markdown backslash escapes
				`\*`, `*`,
				`\_`, `_`,
				`\[`, `[`,
				`\]`, `]`,
				`\#`, `#`,
				"\\`", "`",
				`\&`, `&`,
				// JSON unicode escapes that LLMs double-escape
				`\u0026`, `&`,
				`\u003c`, `<`,
				`\u003e`, `>`,
				`\u0022`, `"`,
				`\u0027`, `'`,
			)
			normalizedOldStr := normalizer.Replace(input.OldStr)
			normalizedDoc := normalizer.Replace(documentMarkdown)
			index = strings.Index(normalizedDoc, normalizedOldStr)
			if index != -1 {
				documentMarkdown = normalizedDoc
				input.OldStr = normalizedOldStr
				log.Printf("   🔄 Matched after normalizing markdown escapes")
				// #region agent log
				debugNorm := fmt.Sprintf(`{"location":"writing.go:edit_text_norm","message":"MATCHED via escape normalization","data":{"index":%d},"timestamp":%d,"hypothesisId":"H3"}`, index, time.Now().UnixMilli())
				if f, err := os.OpenFile("/Users/kgil/Git/blogs/blog-agent-go/.cursor/debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil { f.WriteString(debugNorm + "\n"); f.Close() }
				// #endregion
			}
		}

		// Second fallback: also collapse repeated whitespace (spaces, tabs) to single space
		if index == -1 {
			collapseWS := func(s string) string {
				// Normalize markdown escapes and JSON unicode escapes first
				n := strings.NewReplacer(`\*`, `*`, `\_`, `_`, `\[`, `[`, `\]`, `]`, `\#`, `#`, "\\`", "`", `\&`, `&`, `\u0026`, `&`, `\u003c`, `<`, `\u003e`, `>`, `\u0022`, `"`, `\u0027`, `'`).Replace(s)
				// Collapse runs of whitespace (but preserve newlines)
				var b strings.Builder
				prevSpace := false
				for _, r := range n {
					if r == ' ' || r == '\t' {
						if !prevSpace { b.WriteRune(' ') }
						prevSpace = true
					} else {
						b.WriteRune(r)
						prevSpace = false
					}
				}
				return b.String()
			}
			wsOldStr := collapseWS(input.OldStr)
			wsDoc := collapseWS(documentMarkdown)
			index = strings.Index(wsDoc, wsOldStr)
			if index != -1 {
				documentMarkdown = wsDoc
				input.OldStr = wsOldStr
				log.Printf("   🔄 Matched after whitespace normalization")
				// #region agent log
				debugWS := fmt.Sprintf(`{"location":"writing.go:edit_text_ws","message":"MATCHED via whitespace normalization","data":{"index":%d},"timestamp":%d,"hypothesisId":"H3"}`, index, time.Now().UnixMilli())
				if f, err := os.OpenFile("/Users/kgil/Git/blogs/blog-agent-go/.cursor/debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil { f.WriteString(debugWS + "\n"); f.Close() }
				// #endregion
			}
		}

		// Third fallback: fuzzy patch matching using diffmatchpatch
		if index == -1 {
			log.Printf("   🔄 Trying fuzzy patch match...")
			fuzzyDmp := diffmatchpatch.New()
			fuzzyDmp.MatchThreshold = 0.3 // Allow 30% character differences
			fuzzyDmp.MatchDistance = 1000  // Search across a wide range
			fuzzyDmp.PatchDeleteThreshold = 0.4

			// Create a patch from old_str -> new_str
			patches := fuzzyDmp.PatchMake(input.OldStr, input.NewStr)
			// Apply the patch to the full document with fuzzy matching
			applied, results := fuzzyDmp.PatchApply(patches, documentMarkdown)
			anyApplied := false
			for _, r := range results {
				if r { anyApplied = true; break }
			}
			if anyApplied && applied != documentMarkdown {
				newMarkdown = applied
				log.Printf("   🔄 Fuzzy patch applied successfully (%d/%d hunks)", countTrue(results), len(results))
				// #region agent log
				debugFuzzy := fmt.Sprintf(`{"location":"writing.go:edit_text_fuzzy","message":"MATCHED via fuzzy patch","data":{"hunksApplied":%d,"hunksTotal":%d,"newLen":%d},"timestamp":%d,"hypothesisId":"H_fuzzy"}`,
					countTrue(results), len(results), len(newMarkdown), time.Now().UnixMilli())
				if f, err := os.OpenFile("/Users/kgil/Git/blogs/blog-agent-go/.cursor/debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil { f.WriteString(debugFuzzy + "\n"); f.Close() }
				// #endregion
				// Skip the exact match path below -- we already have newMarkdown
				index = 0 // sentinel: mark as found so we skip the error path
			}
		}

		if index == -1 {
			log.Printf("   ❌ old_str not found even with fuzzy matching")
			// #region agent log
			debugFail := fmt.Sprintf(`{"location":"writing.go:edit_text_fail","message":"ALL MATCH STRATEGIES FAILED","data":{"oldStrLen":%d,"docLen":%d},"timestamp":%d,"hypothesisId":"H_fail"}`,
				len(input.OldStr), len(documentMarkdown), time.Now().UnixMilli())
			if f, err := os.OpenFile("/Users/kgil/Git/blogs/blog-agent-go/.cursor/debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil { f.WriteString(debugFail + "\n"); f.Close() }
			// #endregion

			// Return error but include the proposed edit data so the DiffArtifact can display it
			result := map[string]interface{}{
				"old_str":   input.OldStr,
				"new_str":   input.NewStr,
				"reason":    input.Reason,
				"tool_name": "edit_text",
				"is_error":  true,
				"error":     "Could not locate the text to edit. The text may contain special characters that were modified during formatting.",
			}
			resultJSON, _ := json.Marshal(result)
			return ToolResponse{
				Type:    ToolResponseTypeText,
				Content: string(resultJSON),
				Result:  result,
				IsError: true,
				Artifact: &ArtifactHint{
					Type: ArtifactHintTypeDiff,
					Data: map[string]interface{}{
						"original": input.OldStr,
						"proposed": input.NewStr,
						"reason":   input.Reason,
					},
				},
			}, nil
		}

		if index != -1 && newMarkdown == "" {
			// Exact/normalized match succeeded -- apply via string replacement
			// (fuzzy path already set newMarkdown, so skip this if newMarkdown is set)
			lastIndex := strings.LastIndex(documentMarkdown, input.OldStr)
			if index != lastIndex {
				log.Printf("   ❌ old_str appears multiple times in document")
				return NewTextErrorResponse("old_str appears multiple times in the document. Include more surrounding context to make it unique."), nil
			}

			newMarkdown = documentMarkdown[:index] + input.NewStr + documentMarkdown[index+len(input.OldStr):]
			log.Printf("   ✅ Edit applied to document markdown (new length: %d)", len(newMarkdown))
			// #region agent log
			debugOk := fmt.Sprintf(`{"location":"writing.go:edit_text_ok","message":"MATCH SUCCESS","data":{"index":%d,"oldStrLen":%d},"timestamp":%d,"hypothesisId":"H1"}`,
				index, len(input.OldStr), time.Now().UnixMilli())
			if f, err := os.OpenFile("/Users/kgil/Git/blogs/blog-agent-go/.cursor/debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil { f.WriteString(debugOk + "\n"); f.Close() }
			// #endregion
		}
	} else {
		log.Printf("   ⚠️ No document markdown in context, returning edit without validation")
	}

	// Generate diff for display
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(input.OldStr, input.NewStr, false)

	// Prepare the result
	result := map[string]interface{}{
		"old_str":   input.OldStr,
		"new_str":   input.NewStr,
		"reason":    input.Reason,
		"tool_name": "edit_text",
	}

	// Include the full new markdown if we were able to apply the edit
	if newMarkdown != "" {
		result["new_markdown"] = newMarkdown
	}

	result["patch"] = map[string]interface{}{
		"summary": map[string]interface{}{
			"additions": countDiffType(diffs, diffmatchpatch.DiffInsert),
			"deletions": countDiffType(diffs, diffmatchpatch.DiffDelete),
			"unchanged": countDiffType(diffs, diffmatchpatch.DiffEqual),
		},
	}

	// Create artifact hint for diff display
	artifactData := map[string]interface{}{
		"original": input.OldStr,
		"proposed": input.NewStr,
		"reason":   input.Reason,
	}

	resultJSON, _ := json.Marshal(result)
	return ToolResponse{
		Type:    ToolResponseTypeText,
		Content: string(resultJSON),
		Result:  result,
		Artifact: &ArtifactHint{
			Type: ArtifactHintTypeDiff,
			Data: artifactData,
		},
	}, nil
}

// Helper function to count characters by diff type
func countTrue(results []bool) int {
	count := 0
	for _, r := range results {
		if r { count++ }
	}
	return count
}

func countDiffType(diffs []diffmatchpatch.Diff, diffType diffmatchpatch.Operation) int {
	count := 0
	for _, diff := range diffs {
		if diff.Type == diffType {
			count += len(diff.Text)
		}
	}
	return count
}

// AddContextFromSourcesTool finds and adds relevant context from sources to enhance content
type AddContextFromSourcesTool struct {
	sourceService ArticleSourceService
}

func NewAddContextFromSourcesTool(sourceService ArticleSourceService) *AddContextFromSourcesTool {
	return &AddContextFromSourcesTool{
		sourceService: sourceService,
	}
}

func (t *AddContextFromSourcesTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "add_context_from_sources",
		Description: "Find relevant sources and add contextual information to enhance the current document content",
		Parameters: map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "The search query to find relevant sources (topics, keywords from the document)",
			},
			"current_content": map[string]any{
				"type":        "string",
				"description": "Current document content to provide context for search",
			},
			"limit": map[string]any{
				"type":        []string{"number", "null"},
				"description": "Maximum number of relevant sources to return (default: 5)",
			},
		},
		Required: []string{"query", "current_content"},
	}
}

func (t *AddContextFromSourcesTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
	var input struct {
		Query          string `json:"query"`
		CurrentContent string `json:"current_content"`
		Limit          int    `json:"limit"`
	}

	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return NewTextErrorResponse("Invalid input format"), err
	}

	if input.Query == "" || input.CurrentContent == "" {
		return NewTextErrorResponse("query and current_content are required"), fmt.Errorf("query and current_content are required")
	}

	if input.Limit <= 0 {
		input.Limit = 5
	}

	log.Printf("📚 [AddContextFromSources] Starting context search")
	log.Printf("   📝 Query: %q", input.Query)
	log.Printf("   📄 Current content length: %d characters", len(input.CurrentContent))
	log.Printf("   🎯 Limit: %d", input.Limit)

	// Get article ID from context
	articleIDStr := GetArticleIDFromContext(ctx)
	if articleIDStr == "" {
		log.Printf("📚 [AddContextFromSources] WARNING: No article ID in context")
		result := map[string]interface{}{
			"relevant_sources": []map[string]interface{}{},
			"context_added":    false,
			"query":            input.Query,
			"tool_name":        "add_context_from_sources",
			"warning":          "No article ID available - cannot search for sources",
		}
		resultJSON, _ := json.Marshal(result)
		return NewTextResponse(string(resultJSON)), nil
	}

	articleID, err := uuid.Parse(articleIDStr)
	if err != nil {
		return NewTextErrorResponse("Invalid article ID"), fmt.Errorf("invalid article ID: %w", err)
	}

	// Search for similar sources
	sources, err := t.sourceService.SearchSimilarSources(ctx, articleID, input.Query, input.Limit)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("Failed to search sources: %v", err)), err
	}

	log.Printf("📚 [AddContextFromSources] Found %d sources", len(sources))

	// Convert sources to response format with chunking
	var relevantSources []map[string]interface{}
	for i, source := range sources {
		chunks := t.chunkText(source.Content, 1200)
		relevantChunks := t.findMostRelevantChunks(chunks, input.Query, 2)

		for j, chunk := range relevantChunks {
			sourceData := map[string]interface{}{
				"source_title": source.Title,
				"source_url":   source.URL,
				"text_chunk":   chunk.Text,
				"source_type":  source.SourceType,
				"chunk_score":  chunk.Score,
				"chunk_index":  j + 1,
				"source_index": i + 1,
			}
			relevantSources = append(relevantSources, sourceData)
		}
	}

	result := map[string]interface{}{
		"relevant_sources": relevantSources,
		"context_added":    len(relevantSources) > 0,
		"query":            input.Query,
		"total_sources":    len(sources),
		"total_chunks":     len(relevantSources),
		"tool_name":        "add_context_from_sources",
	}

	log.Printf("📚 [AddContextFromSources] ✅ Returning %d relevant chunks from %d sources", len(relevantSources), len(sources))

	resultJSON, _ := json.Marshal(result)
	return NewTextResponse(string(resultJSON)), nil
}

// chunkText splits text into overlapping chunks for better context preservation
func (t *AddContextFromSourcesTool) chunkText(text string, chunkSize int) []TextChunk {
	if len(text) <= chunkSize {
		return []TextChunk{{Text: text, Index: 0}}
	}

	var chunks []TextChunk
	overlap := chunkSize / 3 // 33% overlap for better context preservation

	for i := 0; i < len(text); i += chunkSize - overlap {
		end := i + chunkSize
		if end > len(text) {
			end = len(text)
		}

		chunk := text[i:end]
		// Try to break at sentence boundaries to avoid cutting words
		if end < len(text) {
			// Look for the last sentence boundary in the last third of the chunk
			searchStart := len(chunk) * 2 / 3
			if searchStart < len(chunk) {
				lastPart := chunk[searchStart:]
				if lastDot := strings.LastIndex(lastPart, "."); lastDot != -1 {
					chunk = chunk[:searchStart+lastDot+1]
				} else if lastQuestion := strings.LastIndex(lastPart, "?"); lastQuestion != -1 {
					chunk = chunk[:searchStart+lastQuestion+1]
				} else if lastExclamation := strings.LastIndex(lastPart, "!"); lastExclamation != -1 {
					chunk = chunk[:searchStart+lastExclamation+1]
				}
			}
		}

		chunks = append(chunks, TextChunk{
			Text:  strings.TrimSpace(chunk),
			Index: len(chunks),
		})

		if end >= len(text) {
			break
		}
	}

	return chunks
}

// findMostRelevantChunks finds the most relevant chunks using simple text similarity
func (t *AddContextFromSourcesTool) findMostRelevantChunks(chunks []TextChunk, query string, maxChunks int) []TextChunk {
	if len(chunks) == 0 {
		return chunks
	}

	// Score each chunk based on keyword overlap with query
	queryWords := t.extractKeywords(strings.ToLower(query))

	for i := range chunks {
		chunks[i].Score = t.calculateRelevanceScore(chunks[i].Text, queryWords)
	}

	// Sort by score (highest first)
	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].Score > chunks[j].Score
	})

	// Return top chunks, but limit to maxChunks
	if len(chunks) > maxChunks {
		chunks = chunks[:maxChunks]
	}

	return chunks
}

// extractKeywords extracts meaningful keywords from a query
func (t *AddContextFromSourcesTool) extractKeywords(text string) []string {
	// Simple keyword extraction - split on spaces and filter common words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
		"be": true, "been": true, "being": true, "have": true, "has": true, "had": true,
		"do": true, "does": true, "did": true, "will": true, "would": true, "could": true,
		"should": true, "may": true, "might": true, "must": true, "can": true,
		"this": true, "that": true, "these": true, "those": true, "i": true, "you": true,
		"he": true, "she": true, "it": true, "we": true, "they": true, "me": true,
		"him": true, "her": true, "us": true, "them": true,
	}

	words := strings.Fields(text)
	var keywords []string

	for _, word := range words {
		// Clean the word
		word = strings.ToLower(strings.Trim(word, ".,!?;:()[]{}\"'"))

		// Skip if empty, too short, or a stop word
		if len(word) < 3 || stopWords[word] {
			continue
		}

		keywords = append(keywords, word)
	}

	return keywords
}

// calculateRelevanceScore calculates a simple relevance score based on keyword frequency
func (t *AddContextFromSourcesTool) calculateRelevanceScore(text string, queryKeywords []string) float64 {
	if len(queryKeywords) == 0 {
		return 0.0
	}

	textLower := strings.ToLower(text)
	textWords := strings.Fields(textLower)
	textWordCount := make(map[string]int)

	// Count word frequencies in text
	for _, word := range textWords {
		word = strings.Trim(word, ".,!?;:()[]{}\"'")
		if len(word) > 2 {
			textWordCount[word]++
		}
	}

	// Calculate score based on keyword matches
	var score float64
	matchedKeywords := 0

	for _, keyword := range queryKeywords {
		if count, exists := textWordCount[keyword]; exists {
			// Use TF-IDF inspired scoring: frequency * log(text_length / keyword_frequency)
			tf := float64(count) / float64(len(textWords))
			idf := math.Log(float64(len(textWords)) / float64(count))
			score += tf * idf
			matchedKeywords++
		}
	}

	// Boost score based on percentage of matched keywords
	keywordCoverage := float64(matchedKeywords) / float64(len(queryKeywords))
	score *= (1.0 + keywordCoverage)

	return score
}

// GenerateTextContentTool generates new text content using LLM
type GenerateTextContentTool struct {
	textGenService TextGenerationService
}

func NewGenerateTextContentTool(textGenService TextGenerationService) *GenerateTextContentTool {
	return &GenerateTextContentTool{
		textGenService: textGenService,
	}
}

func (t *GenerateTextContentTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "generate_text_content",
		Description: "Generate new text content using an LLM, optionally enhanced with contextual sources",
		Parameters: map[string]any{
			"prompt": map[string]any{
				"type":        "string",
				"description": "The generation prompt or instructions for the LLM",
			},
			"context_sources": map[string]any{
				"type":        []string{"array", "null"},
				"description": "Optional: Array of relevant source chunks to provide context",
				"items": map[string]any{
					"type": "object",
				},
			},
			"original_content": map[string]any{
				"type":        []string{"string", "null"},
				"description": "Optional: Original content for reference",
			},
		},
		Required: []string{"prompt"},
	}
}

func (t *GenerateTextContentTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
	var input struct {
		Prompt          string                   `json:"prompt"`
		ContextSources  []map[string]interface{} `json:"context_sources"`
		OriginalContent string                   `json:"original_content"`
	}

	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return NewTextErrorResponse("Invalid input format"), err
	}

	if input.Prompt == "" {
		return NewTextErrorResponse("prompt is required"), fmt.Errorf("prompt is required")
	}

	log.Printf("✍️ [GenerateTextContent] Starting text generation")
	log.Printf("   📝 Prompt length: %d characters", len(input.Prompt))
	log.Printf("   📚 Context sources: %d", len(input.ContextSources))
	log.Printf("   📄 Original content: %d characters", len(input.OriginalContent))

	// Build enhanced prompt with context sources
	enhancedPrompt := input.Prompt

	if len(input.ContextSources) > 0 {
		enhancedPrompt += "\n\n--- Relevant Context Sources ---\n"
		for i, source := range input.ContextSources {
			if title, ok := source["source_title"].(string); ok {
				enhancedPrompt += fmt.Sprintf("\n%d. %s", i+1, title)
			}
			if url, ok := source["source_url"].(string); ok {
				enhancedPrompt += fmt.Sprintf(" (%s)", url)
			}
			if chunk, ok := source["text_chunk"].(string); ok {
				enhancedPrompt += fmt.Sprintf("\n%s\n", chunk)
			}
		}
		log.Printf("✍️ [GenerateTextContent] Enhanced prompt with %d context sources", len(input.ContextSources))
	}

	if input.OriginalContent != "" {
		enhancedPrompt += "\n\n--- Original Content ---\n" + input.OriginalContent
		log.Printf("✍️ [GenerateTextContent] Added original content as reference")
	}

	// For now, we'll return the enhanced prompt as this tool is meant to be a placeholder
	// In a real implementation, this would call an LLM service
	result := map[string]interface{}{
		"generated_content": enhancedPrompt, // This would be LLM-generated content
		"prompt_used":       input.Prompt,
		"sources_included":  len(input.ContextSources),
		"has_original":      input.OriginalContent != "",
		"tool_name":         "generate_text_content",
		"generation_method": "enhanced_prompt", // Indicates this is using context enhancement
	}

	log.Printf("✍️ [GenerateTextContent] ✅ Generated content with context enhancement")

	// Create artifact hint for content generation display
	artifactData := map[string]interface{}{
		"generated_content": enhancedPrompt,
		"prompt":            input.Prompt,
		"section_type":      "generated",
	}

	resultJSON, _ := json.Marshal(result)
	return ToolResponse{
		Type:    ToolResponseTypeText,
		Content: string(resultJSON),
		Result:  result,
		Artifact: &ArtifactHint{
			Type: ArtifactHintTypeContent,
			Data: artifactData,
		},
	}, nil
}

// GenerateImagePromptTool generates image prompts from content
type GenerateImagePromptTool struct {
	textGenService TextGenerationService
}

func NewGenerateImagePromptTool(textGenService TextGenerationService) *GenerateImagePromptTool {
	return &GenerateImagePromptTool{
		textGenService: textGenService,
	}
}

func (t *GenerateImagePromptTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "generate_image_prompt",
		Description: "Generate an image prompt based on document content",
		Parameters: map[string]any{
			"content": map[string]any{
				"type":        "string",
				"description": "The document content to generate image prompt for",
			},
		},
		Required: []string{"content"},
	}
}

func (t *GenerateImagePromptTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
	var input struct {
		Content string `json:"content"`
	}

	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return NewTextErrorResponse("Invalid input format"), err
	}

	if input.Content == "" {
		return NewTextErrorResponse("content is required"), fmt.Errorf("content is required")
	}

	prompt, err := t.textGenService.GenerateImagePrompt(ctx, input.Content)
	if err != nil {
		return NewTextErrorResponse(fmt.Sprintf("Failed to generate image prompt: %v", err)), err
	}

	result := map[string]interface{}{
		"prompt":    prompt,
		"tool_name": "generate_image_prompt",
	}

	// Create artifact hint for image prompt display
	artifactData := map[string]interface{}{
		"prompt":       prompt,
		"content_hint": input.Content[:min(200, len(input.Content))],
	}

	resultJSON, _ := json.Marshal(result)
	return ToolResponse{
		Type:    ToolResponseTypeText,
		Content: string(resultJSON),
		Result:  result,
		Artifact: &ArtifactHint{
			Type: ArtifactHintTypeImagePrompt,
			Data: artifactData,
		},
	}, nil
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
