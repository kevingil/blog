package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"

	"blog-agent-go/backend/internal/models"

	"github.com/google/uuid"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// ArticleSourceService interface for source operations - using the real service directly
type ArticleSourceService interface {
	SearchSimilarSources(ctx context.Context, articleID uuid.UUID, query string, limit int) ([]*models.ArticleSource, error)
}

// TextGenerationService interface for text generation operations
type TextGenerationService interface {
	GenerateImagePrompt(ctx context.Context, content string) (string, error)
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
				"type":        "number",
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

	// Convert models.ArticleSource to response format with text chunking
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

	resultJSON, _ := json.Marshal(result)
	return NewTextResponse(string(resultJSON)), nil
}

// RewriteDocumentTool rewrites the entire document
type RewriteDocumentTool struct {
	textGenService TextGenerationService
	sourceService  ArticleSourceService
}

func NewRewriteDocumentTool(textGenService TextGenerationService, sourceService ArticleSourceService) *RewriteDocumentTool {
	return &RewriteDocumentTool{
		textGenService: textGenService,
		sourceService:  sourceService,
	}
}

func (t *RewriteDocumentTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "rewrite_document",
		Description: "Completely rewrite or significantly edit the document content with diff generation support. CRITICAL: Write like a human, not an AI. Avoid AI writing patterns: no puffery ('breathtaking', 'nestled'), no symbolic importance phrases ('stands as a testament'), no editorializing ('it's important to note'), no superficial analyses with -ing phrases, no overused conjunctions, no section summaries, no negative parallelisms, no excessive em dashes, use sentence case headings, avoid vague attributions, write naturally with varied structures and concrete details.",
		Parameters: map[string]any{
			"new_content": map[string]any{
				"type":        "string",
				"description": "The new document content in markdown format",
			},
			"reason": map[string]any{
				"type":        "string",
				"description": "Brief explanation of the changes made",
			},
			"original_content": map[string]any{
				"type":        "string",
				"description": "Optional: Original document content for generating diff patches. When provided, enables diff preview functionality.",
			},
		},
		Required: []string{"new_content", "reason"},
	}
}

func (t *RewriteDocumentTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
	var input struct {
		NewContent      string `json:"new_content"`
		Reason          string `json:"reason"`
		OriginalContent string `json:"original_content,omitempty"` // Optional for diff generation
	}

	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return NewTextErrorResponse("Invalid input format"), err
	}

	if input.NewContent == "" {
		return NewTextErrorResponse("new_content is required"), fmt.Errorf("new_content is required")
	}

	log.Printf("📝 [RewriteDocument] Processing document rewrite")
	log.Printf("   📄 New content length: %d characters", len(input.NewContent))
	log.Printf("   📝 Reason: %q", input.Reason)

	result := map[string]interface{}{
		"new_content": input.NewContent,
		"reason":      input.Reason,
		"tool_name":   "rewrite_document",
		"edit_type":   "rewrite",
	}

	// If original content is provided, generate diff patch like edit_text tool
	if input.OriginalContent != "" {
		log.Printf("📝 [RewriteDocument] Generating diff patch")
		
		// Generate unified diff patch using diffmatchpatch
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(input.OriginalContent, input.NewContent, false)
		patch := dmp.PatchMake(input.OriginalContent, diffs)
		patchText := dmp.PatchToText(patch)

		// Add patch information to result
		result["original_content"] = input.OriginalContent
		result["patch"] = map[string]interface{}{
			"unified_diff": patchText,
			"diffs":        diffs,
			"summary": map[string]interface{}{
				"additions": countDiffType(diffs, diffmatchpatch.DiffInsert),
				"deletions": countDiffType(diffs, diffmatchpatch.DiffDelete),
				"unchanged": countDiffType(diffs, diffmatchpatch.DiffEqual),
			},
		}
		
		log.Printf("📝 [RewriteDocument] ✅ Diff patch generated - Additions: %d, Deletions: %d", 
			countDiffType(diffs, diffmatchpatch.DiffInsert),
			countDiffType(diffs, diffmatchpatch.DiffDelete))
	}

	log.Printf("📝 [RewriteDocument] ✅ Document rewrite completed")

	resultJSON, _ := json.Marshal(result)
	return NewTextResponse(string(resultJSON)), nil
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
		Name:        "edit_text",
		Description: "Edit specific text in the document while preserving the rest. Use this for targeted edits, improvements, or changes to specific sections. IMPORTANT: Write like a human - avoid AI patterns like puffery words, symbolic importance phrases, editorializing, superficial analyses, overused conjunctions, section summaries, and negative parallelisms. Use natural, varied sentence structures.",
		Parameters: map[string]any{
			"original_text": map[string]any{
				"type":        "string",
				"description": "The exact text to find and replace in the document",
			},
			"new_text": map[string]any{
				"type":        "string",
				"description": "The new text to replace the original text with",
			},
			"reason": map[string]any{
				"type":        "string",
				"description": "Brief explanation of why this edit is being made",
			},
		},
		Required: []string{"original_text", "new_text", "reason"},
	}
}

func (t *EditTextTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
	var input struct {
		OriginalText string `json:"original_text"`
		NewText      string `json:"new_text"`
		Reason       string `json:"reason"`
	}

	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return NewTextErrorResponse("Invalid input format"), err
	}

	if input.OriginalText == "" || input.NewText == "" {
		return NewTextErrorResponse("original_text and new_text are required"), fmt.Errorf("original_text and new_text are required")
	}

	// Generate unified diff patch using diffmatchpatch
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(input.OriginalText, input.NewText, false)
	patch := dmp.PatchMake(input.OriginalText, diffs)
	patchText := dmp.PatchToText(patch)

	// Prepare the result with patch information
	result := map[string]interface{}{
		"original_text": input.OriginalText,
		"new_text":      input.NewText,
		"reason":        input.Reason,
		"edit_type":     "patch",
		"tool_name":     "edit_text",
		"patch": map[string]interface{}{
			"unified_diff": patchText,
			"diffs":        diffs,
			"summary": map[string]interface{}{
				"additions": countDiffType(diffs, diffmatchpatch.DiffInsert),
				"deletions": countDiffType(diffs, diffmatchpatch.DiffDelete),
				"unchanged": countDiffType(diffs, diffmatchpatch.DiffEqual),
			},
		},
	}

	resultJSON, _ := json.Marshal(result)
	return NewTextResponse(string(resultJSON)), nil
}

// Helper function to count characters by diff type
func countDiffType(diffs []diffmatchpatch.Diff, diffType diffmatchpatch.Operation) int {
	count := 0
	for _, diff := range diffs {
		if diff.Type == diffType {
			count += len(diff.Text)
		}
	}
	return count
}

// AnalyzeDocumentTool analyzes document and provides suggestions
type AnalyzeDocumentTool struct{}

func NewAnalyzeDocumentTool() *AnalyzeDocumentTool {
	return &AnalyzeDocumentTool{}
}

func (t *AnalyzeDocumentTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "analyze_document",
		Description: "Analyze document and provide improvement suggestions. Can focus on specific areas or provide general analysis.",
		Parameters: map[string]any{
			"focus_area": map[string]any{
				"type":        "string",
				"description": "Optional: What aspect to focus on (structure, clarity, engagement, grammar, flow, technical_accuracy). If not provided, will analyze overall document quality.",
			},
			"user_request": map[string]any{
				"type":        "string",
				"description": "The user's original request to help understand what they want to improve",
			},
		},
		Required: []string{"user_request"},
	}
}

func (t *AnalyzeDocumentTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
	var input struct {
		FocusArea   string `json:"focus_area"`
		UserRequest string `json:"user_request"`
	}

	if err := json.Unmarshal([]byte(params.Input), &input); err != nil {
		return NewTextErrorResponse("Invalid input format"), err
	}

	if input.UserRequest == "" {
		return NewTextErrorResponse("user_request is required"), fmt.Errorf("user_request is required")
	}

	// Infer focus area from user request if not provided
	if input.FocusArea == "" {
		userRequestLower := strings.ToLower(input.UserRequest)
		if strings.Contains(userRequestLower, "engaging") || strings.Contains(userRequestLower, "boring") {
			input.FocusArea = "engagement"
		} else if strings.Contains(userRequestLower, "clear") || strings.Contains(userRequestLower, "confusing") {
			input.FocusArea = "clarity"
		} else if strings.Contains(userRequestLower, "structure") || strings.Contains(userRequestLower, "organize") {
			input.FocusArea = "structure"
		} else if strings.Contains(userRequestLower, "grammar") || strings.Contains(userRequestLower, "spelling") {
			input.FocusArea = "grammar"
		} else {
			input.FocusArea = "overall"
		}
	}

	result := map[string]interface{}{
		"focus_area":    input.FocusArea,
		"user_request":  input.UserRequest,
		"analysis_done": true,
		"tool_name":     "analyze_document",
	}

	resultJSON, _ := json.Marshal(result)
	return NewTextResponse(string(resultJSON)), nil
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
				"type":        "number",
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
			"relevant_sources":   []map[string]interface{}{},
			"context_added":      false,
			"query":              input.Query,
			"tool_name":          "add_context_from_sources",
			"warning":            "No article ID available - cannot search for sources",
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
				"type":        "array",
				"description": "Optional: Array of relevant source chunks to provide context",
				"items": map[string]any{
					"type": "object",
				},
			},
			"original_content": map[string]any{
				"type":        "string",
				"description": "Optional: Original content for reference",
			},
		},
		Required: []string{"prompt"},
	}
}

func (t *GenerateTextContentTool) Run(ctx context.Context, params ToolCall) (ToolResponse, error) {
	var input struct {
		Prompt         string                   `json:"prompt"`
		ContextSources []map[string]interface{} `json:"context_sources"`
		OriginalContent string                  `json:"original_content"`
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
		"generated_content":  enhancedPrompt, // This would be LLM-generated content
		"prompt_used":        input.Prompt,
		"sources_included":   len(input.ContextSources),
		"has_original":       input.OriginalContent != "",
		"tool_name":          "generate_text_content",
		"generation_method":  "enhanced_prompt", // Indicates this is using context enhancement
	}

	log.Printf("✍️ [GenerateTextContent] ✅ Generated content with context enhancement")

	resultJSON, _ := json.Marshal(result)
	return NewTextResponse(string(resultJSON)), nil
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

	resultJSON, _ := json.Marshal(result)
	return NewTextResponse(string(resultJSON)), nil
}
