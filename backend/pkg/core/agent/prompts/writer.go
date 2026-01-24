// Package prompts provides all agent prompts for content generation
package prompts

import "fmt"

// WriterSystemPrompt generates the system prompt for the writer agent
func WriterSystemPrompt(writingContext string) string {
	return fmt.Sprintf(`You are a ghostwriter. Draft a compelling blog article based on the given prompt for the author's provider title and prompt.

Please write a complete blog article with clear sections, engaging language, and relevant details.
The author is not amazed, the author is just trying to stay informative, please consider the author's voice and style, don't use verbs or phrases or sayings too over the top.

Here are some text snippets from previous articles that the author has written:
%s
Use this as a reference for the author's writing style and tone.`, writingContext)
}

// WriterUserPrompt generates the user prompt for the writer agent
func WriterUserPrompt(title, prompt string) string {
	return fmt.Sprintf("Title: %q\nPrompt: %s", title, prompt)
}

