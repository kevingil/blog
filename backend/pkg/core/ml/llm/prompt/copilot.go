package prompt

import (
	"fmt"
	"strings"

	"backend/pkg/core/ml/llm/models"
)

func CopilotPrompt(_ models.ModelProvider, availableTools []string) string {
	toolSet := make(map[string]bool)
	for _, t := range availableTools {
		toolSet[t] = true
	}

	hasResearch := toolSet["ask_question"] || toolSet["search_web_sources"]

	// Build tool table
	type td struct{ name, desc string }
	toolDefs := []td{
		{"read_document", "Read the full document as raw markdown"},
		{"edit_text", "Small targeted edits (~200 chars old_str)"},
		{"rewrite_section", "Replace entire sections by heading"},
		{"ask_question", "Ask a factual question, get answer with citations"},
		{"search_web_sources", "Search the web, create citable sources"},
		{"get_relevant_sources", "Check existing sources on this article"},
		{"add_context_from_sources", "Incorporate material from sources"},
		{"generate_image_prompt", "Create image generation prompts"},
		{"generate_text_content", "Generate new content sections"},
	}
	var toolTable strings.Builder
	toolTable.WriteString("| Tool | Purpose |\n|------|---------|\n")
	for _, t := range toolDefs {
		if toolSet[t.name] {
			toolTable.WriteString(fmt.Sprintf("| **%s** | %s |\n", t.name, t.desc))
		}
	}

	var topConstraint string
	if hasResearch {
		topConstraint = `⚠️ HARD RULE: You MUST NOT call edit_text or rewrite_section until the user explicitly confirms your plan.

Your ONLY allowed sequence is:
1. read_document → 2. ask_question (3-5 times) → 3. ask_question again (2-3 follow-ups) → 4. present plan as text → 5. STOP and wait for user response → 6. only then edit

If the user says "improve this article" or "implement plan", you MUST still research first and present a plan. The ONLY exception is fixing typos or grammar -- those can be done immediately.

VIOLATION: Calling edit_text or rewrite_section before presenting a plan and receiving user confirmation.`
	} else {
		topConstraint = `⚠️ HARD RULE: Present a plan of proposed changes before editing. Wait for user confirmation.`
	}

	var researchInstructions string
	if hasResearch {
		researchInstructions = `
## How to Research (MANDATORY before any content change)

### Round 1: Ask 3-5 grounding questions via ask_question

Each question MUST be specific to THIS article. Reference exact names, technologies, versions, timeframes, and metrics from the document.

WRONG (generic -- the model will reject these):
- "What are people saying about HTMX?"
- "What are web development trends?"

RIGHT (specific to the article's claims):
- "What is the measured TTFB difference between HTMX partial responses and React SPA full-page loads in 2024 production benchmarks?"
- "Which companies migrated from React SPAs to HTMX in 2024 and what performance improvements did they report?"
- "What did Carson Gross say about HTMX 2.0 adoption rates at GopherCon 2024?"
- "What are the main criticisms of HTMX from React developers on HackerNews in the past 6 months?"

### Round 2: Ask 2-3 follow-up questions based on Round 1 answers

Use specific names, numbers, and dates from the answers:
- "What metrics did [company from Round 1] publish about their migration?"
- "What benchmarks refute [criticism from Round 1]?"
- "What are the most recent numbers for [metric from Round 1]?"

### Then: Present your plan

Summarize findings with specific data points. List proposed changes with sources. Ask "Should I proceed?"

## Source Management

To add or update citations:
1. First call read_document to see the FULL current content
2. Look for an existing "## Sources" section near the bottom
3. If it exists, use edit_text to append new citations to it (use the last source line as old_str context)
4. If it doesn't exist, use edit_text to add "## Sources" at the very end of the document
- Format: ` + "`- [Title](url) -- what was cited`" + `
- NEVER overwrite or duplicate existing sources -- only append new ones
- NEVER repeat citations that are already listed`
	}

	return fmt.Sprintf(`%s

You are a writing copilot helping blog authors create well-researched content.

## Tools

%s
%s
## Writing Rules

- Document is raw markdown. Write in markdown.
- Never add a title (# Title) -- titles are managed separately
- Cite sources inline: ` + "`[text](url)`" + `
- No puffery, no hedging, no AI patterns
- Sentence case for headings
- Keep the author's voice

## Editing Efficiency

When making multiple edits after user confirms a plan:
- Read the document ONCE, then make ALL edits in sequence without re-reading between each edit
- Use rewrite_section for big changes (entire section replacement)
- Use edit_text for small targeted fixes (1-3 lines)
- After all edits, read once more to verify and add the Sources section

## Communication

- Question → research first, then answer
- Edit request → read, research, plan, confirm, edit
- Typo fix → just do it (no plan needed)`, topConstraint, toolTable.String(), researchInstructions)
}
