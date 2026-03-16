package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"

	coreSource "backend/pkg/core/source"
	"backend/pkg/types"

	"github.com/google/uuid"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// ArticleSourceService interface for source operations - using the real service directly
type ArticleSourceService interface {
	GetByArticleID(ctx context.Context, articleID uuid.UUID) ([]types.Source, error)
	SearchSimilar(ctx context.Context, articleID uuid.UUID, query string, limit int) ([]types.Source, error)
	UpsertAgentResource(ctx context.Context, req coreSource.AgentResourceSelection) (*types.Source, error)
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
		Name:        "read_document",
		Description: `Read the full document with line numbers. Use line numbers to reference content for replace_lines. The "sections" array shows each heading with its line number.`,
		Parameters:  map[string]any{},
		Required:    []string{},
	}
}

func (t *ReadDocumentTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
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

	// Build numbered content (line numbers for LLM reference)
	numbered := make([]string, totalLines)
	for i, line := range lines {
		numbered[i] = fmt.Sprintf("%4d| %s", i+1, line)
	}
	numberedContent := strings.Join(numbered, "\n")

	// Build section map from headings
	type sectionInfo struct {
		Heading string `json:"heading"`
		Line    int    `json:"line"`
		Level   int    `json:"level"`
	}
	var sections []sectionInfo
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			level := 0
			for _, ch := range trimmed {
				if ch == '#' {
					level++
				} else {
					break
				}
			}
			if level > 0 {
				sections = append(sections, sectionInfo{Heading: trimmed, Line: i + 1, Level: level})
			}
		}
	}

	log.Printf("📖 [ReadDocument] Returning full document (%d lines, %d chars, %d sections)", totalLines, len(docContent), len(sections))

	result := map[string]interface{}{
		"content":     numberedContent,
		"total_lines": totalLines,
		"total_chars": len(docContent),
		"sections":    sections,
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
	sources, err := t.sourceService.SearchSimilar(ctx, articleID, input.Query, input.Limit)
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
		chunks := chunkText(source.Content, 1200)
		relevantChunks := findMostRelevantChunks(chunks, input.Query, 2)

		log.Printf("   🧩 Generated %d chunks, selected %d most relevant", len(chunks), len(relevantChunks))

		// Add each relevant chunk as a separate source entry
		for j, chunk := range relevantChunks {
			chunkPreview := chunk.Text
			if len(chunkPreview) > 200 {
				chunkPreview = chunkPreview[:200] + "..."
			}

			log.Printf("   📝 Chunk #%d (score: %.3f, length: %d chars): %q", j+1, chunk.Score, len(chunk.Text), chunkPreview)

			sourceData := map[string]interface{}{
				"source_id":    source.ID.String(),
				"source_title": source.Title,
				"source_url":   source.URL,
				"text_chunk":   chunk.Text,
				"excerpt_text": chunk.Text,
				"source_type":  source.SourceType,
				"chunk_score":  chunk.Score,
				"chunk_index":  chunk.Index,
				"excerpt_id":   fmt.Sprintf("%s:%d", source.ID.String(), chunk.Index),
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
		"source_inventory": BuildSourceContextResources(sources),
		"query":            input.Query,
		"total_found":      len(relevantSources),
		"tool_name":        "get_relevant_sources",
	}

	log.Printf("🔍 [GetRelevantSources] ✅ Returning %d relevant chunks from %d sources", len(relevantSources), len(sources))

	// Create artifact hint for sources display
	inventory := result["source_inventory"]
	artifactData := map[string]interface{}{
		"sources":          relevantSources,
		"source_inventory": inventory,
		"query":            input.Query,
		"total_found":      len(relevantSources),
		"inventory_count":  len(sources),
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
func chunkText(text string, chunkSize int) []TextChunk {
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
func findMostRelevantChunks(chunks []TextChunk, query string, maxChunks int) []TextChunk {
	if len(chunks) == 0 {
		return chunks
	}

	// Score each chunk based on keyword overlap with query
	queryWords := extractKeywords(strings.ToLower(query))

	for i := range chunks {
		chunks[i].Score = calculateRelevanceScore(chunks[i].Text, queryWords)
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
func extractKeywords(text string) []string {
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
func calculateRelevanceScore(text string, queryKeywords []string) float64 {
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

// ReplaceLinesTool replaces specific lines in the document by line number.
// Line numbers come from read_document's numbered output.
type ReplaceLinesTool struct {
	draftSaver DraftSaver
}

func NewReplaceLinesTool(draftSaver DraftSaver) *ReplaceLinesTool {
	return &ReplaceLinesTool{draftSaver: draftSaver}
}

func (t *ReplaceLinesTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "replace_lines",
		Description: `Replace lines in the document by line number. Use read_document to see line numbers and section boundaries. Works for any change: rewriting sections, fixing typos, adding content, deleting lines. For deletions, omit new_content or set to empty string.`,
		Parameters: map[string]any{
			"start_line": map[string]any{
				"type":        "number",
				"description": "First line to replace (1-indexed, from read_document)",
			},
			"end_line": map[string]any{
				"type":        "number",
				"description": "Last line to replace (inclusive)",
			},
			"new_content": map[string]any{
				"type":        "string",
				"description": "Replacement text. Can be more or fewer lines. Omit or empty to delete lines.",
			},
			"reason": map[string]any{
				"type":        "string",
				"description": "Brief explanation",
			},
		},
		Required: []string{"start_line", "end_line", "reason"},
	}
}

func (t *ReplaceLinesTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
	var input struct {
		StartLine  int    `json:"start_line"`
		EndLine    int    `json:"end_line"`
		NewContent string `json:"new_content"`
		Reason     string `json:"reason"`
	}

	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		log.Printf("   ❌ [ReplaceLines] JSON unmarshal failed: %v", err)
		return NewTextErrorResponse("Invalid input format"), err
	}

	if input.StartLine < 1 || input.EndLine < input.StartLine {
		return NewTextErrorResponse(fmt.Sprintf("Invalid line range: start_line=%d, end_line=%d. Lines are 1-indexed and end_line must be >= start_line.", input.StartLine, input.EndLine)), nil
	}

	log.Printf("✏️ [ReplaceLines] Replacing lines %d-%d", input.StartLine, input.EndLine)
	log.Printf("   📝 Reason: %q", input.Reason)

	documentMarkdown := GetDocumentMarkdownFromContext(ctx)
	if documentMarkdown == "" {
		return NewTextErrorResponse("No document content available"), nil
	}

	lines := strings.Split(documentMarkdown, "\n")
	totalLines := len(lines)

	if input.StartLine > totalLines {
		return NewTextErrorResponse(fmt.Sprintf("start_line %d exceeds document length (%d lines). Call read_document to see current line numbers.", input.StartLine, totalLines)), nil
	}
	if input.EndLine > totalLines {
		input.EndLine = totalLines
	}

	// Extract old content (for before/after diff)
	oldLines := lines[input.StartLine-1 : input.EndLine]
	oldContent := strings.Join(oldLines, "\n")

	// Build new document: before + new_content + after
	before := lines[:input.StartLine-1]
	after := lines[input.EndLine:]

	var newDocLines []string
	newDocLines = append(newDocLines, before...)
	if input.NewContent != "" {
		newDocLines = append(newDocLines, strings.Split(input.NewContent, "\n")...)
	}
	newDocLines = append(newDocLines, after...)

	newMarkdown := strings.Join(newDocLines, "\n")

	log.Printf("   ✅ Replaced lines %d-%d (%d chars -> %d chars)", input.StartLine, input.EndLine, len(oldContent), len(input.NewContent))

	// Update mutable document state and persist
	UpdateDocumentMarkdown(ctx, newMarkdown)
	if t.draftSaver != nil {
		articleID := GetArticleIDFromContext(ctx)
		if articleID != "" {
			if err := t.draftSaver.UpdateDraftContent(ctx, articleID, newMarkdown); err != nil {
				log.Printf("   ⚠️ [ReplaceLines] Failed to persist draft: %v", err)
			} else {
				log.Printf("   💾 [ReplaceLines] Draft persisted to DB")
			}
		}
	}

	result := map[string]interface{}{
		"old_str":      oldContent,
		"new_str":      input.NewContent,
		"new_markdown": newMarkdown,
		"reason":       input.Reason,
		"tool_name":    "replace_lines",
		"start_line":   input.StartLine,
		"end_line":     input.EndLine,
	}

	resultJSON, _ := json.Marshal(result)
	return ToolResponse{
		Type:    ToolResponseTypeText,
		Content: string(resultJSON),
		Result:  result,
		Artifact: &ArtifactHint{
			Type: ArtifactHintTypeDiff,
			Data: map[string]interface{}{
				"original": oldContent,
				"proposed": input.NewContent,
				"reason":   input.Reason,
			},
		},
	}, nil
}

// Helper function to count characters by diff type
func countTrue(results []bool) int {
	count := 0
	for _, r := range results {
		if r {
			count++
		}
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
