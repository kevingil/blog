package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"

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

// ReadDocumentTool allows the agent to read the current document content
type ReadDocumentTool struct{}

func NewReadDocumentTool() *ReadDocumentTool {
	return &ReadDocumentTool{}
}

func (t *ReadDocumentTool) Info() ToolInfo {
	return ToolInfo{
		Name: "read_document",
		Description: `Read the current document content in raw Markdown format.

WHEN TO USE:
- Once at the start of your turn to see the full content
- After multiple edits if you need to verify the current state

The content returned is the EXACT markdown of the document.
You can copy text directly from this output into edit_text old_str.
Read once, then make multiple edits in sequence without re-reading.`,
		Parameters: map[string]any{},
		Required:   []string{},
	}
}

func (t *ReadDocumentTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
	// Get markdown content (preferred) or fall back to HTML
	docContent := GetDocumentMarkdownFromContext(ctx)
	if docContent == "" {
		docContent = GetDocumentHTMLFromContext(ctx)
	}
	if docContent == "" {
		log.Printf("📖 [ReadDocument] ERROR: No document content in context")
		return NewTextErrorResponse("No document content available. The document may be empty or not loaded."), nil
	}

	totalLines := len(strings.Split(docContent, "\n"))
	log.Printf("📖 [ReadDocument] Returning full document (%d lines, %d chars)", totalLines, len(docContent))

	result := map[string]interface{}{
		"content":     docContent,
		"total_lines": totalLines,
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

// EditTextTool edits specific text in the document.
// It updates the mutable DocumentState in context so subsequent read_document calls
// return the post-edit content. If a DraftSaver is provided, it also persists the
// edit to the database immediately (making the backend the source of truth).
type EditTextTool struct {
	draftSaver DraftSaver // nil-safe: skips DB persistence if nil
}

func NewEditTextTool(draftSaver DraftSaver) *EditTextTool {
	return &EditTextTool{draftSaver: draftSaver}
}

func (t *EditTextTool) Info() ToolInfo {
	return ToolInfo{
		Name: "edit_text",
		Description: `Edit specific text in the document using exact string replacement. Best for SMALL, focused changes (1-5 lines).

For replacing an entire section (heading + content), use rewrite_section instead.

CRITICAL REQUIREMENTS:
1. old_str must EXACTLY match text from read_document output (character-for-character)
2. old_str should be SHORT: include only 1-2 lines of context before and after the change
3. Keep old_str under ~200 characters when possible - smaller edits match more reliably
4. The same surrounding context must appear in new_str, with only the changed part modified
5. Read the document once, then make multiple edits in sequence

GOOD EXAMPLE:
  old_str: "page load times.\n\n### Results\n\nAfter running"
  new_str: "page load times.\n\n### Summary\n\nAfter running"

BAD (too large - use rewrite_section for big changes):
  old_str: [entire 500-word section]

Write like a human - avoid puffery, hedging, and AI patterns.`,
		Parameters: map[string]any{
			"old_str": map[string]any{
				"type":        "string",
				"description": "The exact markdown text to find and replace. MUST include 1-2 lines of context before and after the change for uniqueness.",
			},
			"new_str": map[string]any{
				"type":        "string",
				"description": "The replacement markdown text. Must include the same surrounding context as old_str, with only the changed part modified.",
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

	var newMarkdown string
	matchStage := "none"
	if documentMarkdown != "" {
		// Stage 1: Exact match
		index := strings.Index(documentMarkdown, input.OldStr)
		if index != -1 {
			matchStage = "exact"
		}

		// Stage 2: Combined normalization (markdown escapes + unicode + whitespace)
		if index == -1 {
			normalize := func(s string) string {
				// Markdown backslash escapes + JSON unicode escapes
				n := strings.NewReplacer(
					`\*`, `*`, `\_`, `_`, `\[`, `[`, `\]`, `]`, `\#`, `#`, "\\`", "`", `\&`, `&`,
					`\u0026`, `&`, `\u003c`, `<`, `\u003e`, `>`, `\u0022`, `"`, `\u0027`, `'`,
					"\u2014", "-", "\u2013", "-", // em/en dash -> hyphen
					"\u2018", "'", "\u2019", "'", // smart single quotes
					"\u201C", `"`, "\u201D", `"`, // smart double quotes
					"\u2026", "...", "\u00A0", " ", // ellipsis, NBSP
				).Replace(s)
				// Collapse runs of spaces/tabs (preserve newlines)
				var b strings.Builder
				prevSpace := false
				for _, r := range n {
					if r == ' ' || r == '\t' {
						if !prevSpace {
							b.WriteRune(' ')
						}
						prevSpace = true
					} else {
						b.WriteRune(r)
						prevSpace = false
					}
				}
				return b.String()
			}
			normOld := normalize(input.OldStr)
			normDoc := normalize(documentMarkdown)
			index = strings.Index(normDoc, normOld)
			if index != -1 {
				documentMarkdown = normDoc
				input.OldStr = normOld
				input.NewStr = normalize(input.NewStr)
				matchStage = "normalized"
				log.Printf("   🔄 Matched after combined normalization")
			}
		}

		// Stage 3: Fuzzy patch matching using diffmatchpatch
		if index == -1 {
			log.Printf("   🔄 Trying fuzzy patch match...")
			fuzzyDmp := diffmatchpatch.New()
			fuzzyDmp.MatchThreshold = 0.3
			fuzzyDmp.MatchDistance = 1000
			fuzzyDmp.PatchDeleteThreshold = 0.4
			patches := fuzzyDmp.PatchMake(input.OldStr, input.NewStr)
			applied, results := fuzzyDmp.PatchApply(patches, documentMarkdown)
			anyApplied := false
			for _, r := range results {
				if r {
					anyApplied = true
					break
				}
			}
			if anyApplied && applied != documentMarkdown {
				newMarkdown = applied
				matchStage = "fuzzy"
				log.Printf("   🔄 Fuzzy patch applied (%d/%d hunks)", countTrue(results), len(results))
				index = 0 // sentinel so we skip error path
			}
		}

		if matchStage != "exact" {
			log.Printf("   🔍 Match stage: %s (oldStr: %d chars, doc: %d chars)", matchStage, len(input.OldStr), len(documentMarkdown))
		}

		if index == -1 {
			log.Printf("   ❌ old_str not found in document")

			// Build actionable error with document headings
			var headings []string
			for _, line := range strings.Split(documentMarkdown, "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "### ") {
					headings = append(headings, trimmed)
				}
			}
			headingList := strings.Join(headings, ", ")
			if len(headingList) > 300 {
				headingList = headingList[:300] + "..."
			}
			errMsg := fmt.Sprintf("Could not find old_str in the document. "+
				"TIPS: 1) Call read_document to see CURRENT content -- it returns raw markdown you can copy exactly. "+
				"2) Use smaller edits (2-4 lines of context). 3) For large section changes, use rewrite_section instead. "+
				"Document headings: [%s]", headingList)

			result := map[string]interface{}{
				"old_str":   input.OldStr,
				"new_str":   input.NewStr,
				"reason":    input.Reason,
				"tool_name": "edit_text",
				"is_error":  true,
				"error":     errMsg,
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
		}
	} else {
		log.Printf("   ⚠️ No document markdown in context, returning edit without validation")
	}

	// Update the mutable document state so subsequent read_document calls return the
	// post-edit content (solves stale-read during multi-edit turns).
	// Also persist to DB so the backend is the source of truth for draft content.
	if newMarkdown != "" {
		UpdateDocumentMarkdown(ctx, newMarkdown)

		if t.draftSaver != nil {
			articleID := GetArticleIDFromContext(ctx)
			if articleID != "" {
				if err := t.draftSaver.UpdateDraftContent(ctx, articleID, newMarkdown); err != nil {
					log.Printf("   ⚠️ [EditText] Failed to persist draft to DB: %v", err)
				} else {
					log.Printf("   💾 [EditText] Draft content persisted to DB")
				}
			}
		}
	}

	// Generate diff for display
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(input.OldStr, input.NewStr, false)

	// Prepare the result -- includes new_markdown so the frontend can update the editor
	result := map[string]interface{}{
		"old_str":      input.OldStr,
		"new_str":      input.NewStr,
		"reason":       input.Reason,
		"tool_name":    "edit_text",
		"new_markdown": newMarkdown,
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

// RewriteSectionTool replaces an entire document section identified by its heading.
// More reliable than edit_text for large changes because it only requires matching the heading.
type RewriteSectionTool struct {
	draftSaver DraftSaver
}

func NewRewriteSectionTool(draftSaver DraftSaver) *RewriteSectionTool {
	return &RewriteSectionTool{draftSaver: draftSaver}
}

func (t *RewriteSectionTool) Info() ToolInfo {
	return ToolInfo{
		Name: "rewrite_section",
		Description: `Replace an entire section of the document identified by its heading.

WHEN TO USE:
- For replacing, rewriting, or significantly changing a whole section
- When edit_text fails because old_str is too large to match reliably
- For adding new content below an existing section heading

HOW IT WORKS:
- Provide the exact section heading (e.g., "### Best Practices")
- Provide the new content for that section (including the heading itself)
- The tool finds the section boundaries (from heading to next heading of same/higher level)
- Replaces everything in that range with your new content

EXAMPLE:
  section_heading: "### Best Practices"
  new_content: "### Best Practices\n\n| Do | Don't |\n|---|---|\n| Use templates | Inline HTML |"
  reason: "Reformat best practices as a comparison table"

IMPORTANT: new_content MUST start with the section heading.
Write like a human - avoid puffery, hedging, and AI patterns.`,
		Parameters: map[string]any{
			"section_heading": map[string]any{
				"type":        "string",
				"description": "The exact heading text of the section to replace (e.g., '### Best Practices')",
			},
			"new_content": map[string]any{
				"type":        "string",
				"description": "The new markdown content for this section, starting with the heading",
			},
			"reason": map[string]any{
				"type":        "string",
				"description": "Brief explanation of the change",
			},
		},
		Required: []string{"section_heading", "new_content", "reason"},
	}
}

func (t *RewriteSectionTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
	var input struct {
		SectionHeading string `json:"section_heading"`
		NewContent     string `json:"new_content"`
		Reason         string `json:"reason"`
	}

	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return NewTextErrorResponse("Invalid input format"), err
	}

	if input.SectionHeading == "" || input.NewContent == "" {
		return NewTextErrorResponse("section_heading and new_content are required"), fmt.Errorf("section_heading and new_content are required")
	}

	log.Printf("📝 [RewriteSection] Processing section rewrite")
	log.Printf("   📄 Heading: %q", input.SectionHeading)
	log.Printf("   📝 Reason: %q", input.Reason)

	documentMarkdown := GetDocumentMarkdownFromContext(ctx)
	if documentMarkdown == "" {
		return NewTextErrorResponse("No document content available"), nil
	}

	// Determine the heading level (count leading #)
	headingLevel := 0
	for _, ch := range input.SectionHeading {
		if ch == '#' {
			headingLevel++
		} else {
			break
		}
	}
	if headingLevel == 0 {
		return NewTextErrorResponse("section_heading must start with # (e.g., '## Section' or '### Subsection')"), nil
	}

	// Find the section heading in the document
	lines := strings.Split(documentMarkdown, "\n")
	sectionStart := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == strings.TrimSpace(input.SectionHeading) {
			sectionStart = i
			break
		}
	}

	if sectionStart == -1 {
		// Collect available headings for the error message
		var headings []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "### ") || strings.HasPrefix(trimmed, "#### ") {
				headings = append(headings, trimmed)
			}
		}
		headingList := strings.Join(headings, ", ")
		if len(headingList) > 300 {
			headingList = headingList[:300] + "..."
		}

		errMsg := fmt.Sprintf("Section heading %q not found. Use read_document to see current headings. Available: [%s]", input.SectionHeading, headingList)
		return NewTextErrorResponse(errMsg), nil
	}

	// Find the end of the section (next heading of same or higher level, or end of doc)
	sectionEnd := len(lines)
	for i := sectionStart + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "#") {
			// Count the heading level
			level := 0
			for _, ch := range trimmed {
				if ch == '#' {
					level++
				} else {
					break
				}
			}
			if level > 0 && level <= headingLevel {
				sectionEnd = i
				break
			}
		}
	}

	// Build old section content for diff display
	oldSection := strings.Join(lines[sectionStart:sectionEnd], "\n")

	// Build new document: before section + new content + after section
	before := strings.Join(lines[:sectionStart], "\n")
	after := ""
	if sectionEnd < len(lines) {
		after = strings.Join(lines[sectionEnd:], "\n")
	}

	var newMarkdown string
	if before != "" && after != "" {
		newMarkdown = before + "\n" + input.NewContent + "\n" + after
	} else if before != "" {
		newMarkdown = before + "\n" + input.NewContent
	} else if after != "" {
		newMarkdown = input.NewContent + "\n" + after
	} else {
		newMarkdown = input.NewContent
	}

	log.Printf("   ✅ Section replaced: lines %d-%d (%d chars -> %d chars)", sectionStart+1, sectionEnd, len(oldSection), len(input.NewContent))

	// Update mutable document state and persist
	UpdateDocumentMarkdown(ctx, newMarkdown)
	if t.draftSaver != nil {
		articleID := GetArticleIDFromContext(ctx)
		if articleID != "" {
			if err := t.draftSaver.UpdateDraftContent(ctx, articleID, newMarkdown); err != nil {
				log.Printf("   ⚠️ [RewriteSection] Failed to persist draft: %v", err)
			} else {
				log.Printf("   💾 [RewriteSection] Draft persisted to DB")
			}
		}
	}

	// Generate diff
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(oldSection, input.NewContent, false)

	result := map[string]interface{}{
		"old_str":      oldSection,
		"new_str":      input.NewContent,
		"reason":       input.Reason,
		"tool_name":    "rewrite_section",
		"new_markdown": newMarkdown,
	}
	resultJSON, _ := json.Marshal(result)

	return ToolResponse{
		Type:    ToolResponseTypeText,
		Content: string(resultJSON),
		Result:  result,
		Artifact: &ArtifactHint{
			Type: ArtifactHintTypeDiff,
			Data: map[string]interface{}{
				"original": oldSection,
				"proposed": input.NewContent,
				"reason":   input.Reason,
				"diffs":    diffs,
			},
		},
	}, nil
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

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}


// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
