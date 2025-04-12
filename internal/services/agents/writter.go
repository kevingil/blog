package agents

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"blog-agent-go/internal/models"
)

type WriterAgent struct {
	client *anthropic.Client
}

type ArticleChatHistory struct {
	Messages []ArticleChatHistoryMessage `json:"messages"`
	Metadata map[string]interface{}      `json:"metadata"`
}

type ArticleChatHistoryMessage struct {
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	CreatedAt int64                  `json:"created_at"`
	Metadata  map[string]interface{} `json:"metadata"`
}

func NewWriterAgent(apiKey string) *WriterAgent {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &WriterAgent{
		client: &client,
	}
}

const writingContext = `
### How JavaScript Runs in MySQL
Oracle uses PL/SQL as the interface to run JavaScript on MySQL. You can define and save functions that you can later call in your queries. Although some versions of Oracle database already support JavaScript as stored procedures and inline with your query, MySQL only supports JavaScript as saved procedures for the time being. The code runs on the GraalVM runtime, which optimizes your code, converts it to machine code, then runs on the Graal JIT compiler.
### HTMX Frontend
Back on the homepage, we replace the template that was loading the articles with the code below. Using HTMX we easily implement lazy loading by displaying a placeholder as the initial state and calling the /chunks/feed endpoint that uses our new controller to load articles. Once we get a response, HTMX will handle the application state with hx-swap, in this case to replace the placeholder.
### First Day Hike
The hike on the first day did not take long, I started around noon, and finished at 4pm with several water, picture, and food breaks. The first lake is Carr Lake, where most day glampers go, I'm pretty sure I saw a TV setup. Next was, Feely Lake, and Milk Lake, where I stopped for Lunch.
### Running a Perl Script in a Dockerfile
One of the great things about Perl is that it ships with Linux out of the box. It's so well integrated with Unix, it can serve as a wrapper around system tools. Its strong support for text manipulation and data processing makes it very valuable when building distributed systems. When deploying complex Docker applications, there might be some pre-processing during the build process that can take advantage of Perl's many strong suits.
`

func (w *WriterAgent) GenerateArticle(ctx context.Context, prompt, title string, authorID int64) (*models.Article, error) {
	// First draft with writer system message
	writerSystemMsg := fmt.Sprintf(`You are a ghostwriter. Draft a compelling blog article based on the given prompt for the author's provider title and prompt.

	Please write a complete blog article with clear sections, engaging language, and relevant details.
	The author is not amazed, the author is just trying to stay informative, please consider the author's voice and style, don't use verbs or phrases or sayings too over the top.

	Here are some text snippets from previous articles that the author has written:
	%s
	Use this as a reference for the author's writing style and tone.`, writingContext)

	draftMsg, err := w.client.Messages.New(ctx, anthropic.MessageNewParams{
		MaxTokens: 4096,
		Messages: []anthropic.MessageParam{
			{
				Role: anthropic.MessageParamRoleUser,
				Content: []anthropic.ContentBlockParamUnion{
					{
						OfRequestTextBlock: &anthropic.TextBlockParam{Text: writerSystemMsg},
					},
					{
						OfRequestTextBlock: &anthropic.TextBlockParam{Text: fmt.Sprintf("Title: %q\nPrompt: %s", title, prompt)},
					},
				},
			},
		},
		Model: anthropic.ModelClaude3_7SonnetLatest,
	})
	if err != nil {
		return nil, fmt.Errorf("error generating draft: %w", err)
	}

	// Editor refinement
	editorSystemMsg := `You are the Editor. Improve and refine the previously drafted content.
	The blog should be formatted in markdown as follows: 

	[article intro - always start with this, don't add a title or subheader before this, the begginging of the article should be just the intro paragraph]

	### Subheaders
	[article body - there can be multiple subheaders and body sections depending on the article]

	### Conclusion:
	[article conclusion - always end with this]

	If there's code snippets, make sure to format in markdown, if not, don't make up unneeded code snippets.
	Link references can be added at the end of the article in markdown format.
	For listis, use unordered lists with -. 
	- list item
	- list item
	- list item

	Writing style:
	Make it more concise, clear, and engaging but that it's following the author's voice and style
	Make sure the article makes sense and is coherent for a human and that it's not stating the obvious.
	Preserve the main idea and style, but ensure it's polished for publication.
	There's no top players, there's no greatness, there's no revolution, there's just things and information.
	If a brand, thing, or company is mentioned, don't explain it, the author assumes the readers know this information, we are just doing a technical writeup.
	Review and remove uneccessary subheaders or titles as instructed.

	IMPORTANT: 
	the article should not start with a title or subheader, the title is saved somewhere else, we just want the content, start with the first paragraph of the article.
	The voice is the most important, make sure you don't use worlds like ripples or remarkable, just use the words that the author would use.`

	finalMsg, err := w.client.Messages.New(ctx, anthropic.MessageNewParams{
		MaxTokens: 4096,
		Messages: []anthropic.MessageParam{
			{
				Role: anthropic.MessageParamRoleUser,
				Content: []anthropic.ContentBlockParamUnion{
					{
						OfRequestTextBlock: &anthropic.TextBlockParam{Text: editorSystemMsg},
					},
					{
						OfRequestTextBlock: &anthropic.TextBlockParam{Text: draftMsg.Content[0].Text},
					},
				},
			},
		},
		Model: anthropic.ModelClaude3_7SonnetLatest,
	})
	if err != nil {
		return nil, fmt.Errorf("error refining article: %w", err)
	}

	// Create chat history
	chatHistory := ArticleChatHistory{
		Messages: []ArticleChatHistoryMessage{
			{
				Role:      "system",
				Content:   writerSystemMsg,
				CreatedAt: 0,
				Metadata:  map[string]interface{}{},
			},
			{
				Role:      "user",
				Content:   fmt.Sprintf("Title: %q\nPrompt: %s", title, prompt),
				CreatedAt: 1,
				Metadata:  map[string]interface{}{},
			},
			{
				Role:      "assistant",
				Content:   draftMsg.Content[0].Text,
				CreatedAt: 2,
				Metadata:  map[string]interface{}{},
			},
			{
				Role:      "system",
				Content:   editorSystemMsg,
				CreatedAt: 3,
				Metadata:  map[string]interface{}{},
			},
			{
				Role:      "assistant",
				Content:   finalMsg.Content[0].Text,
				CreatedAt: 4,
				Metadata:  map[string]interface{}{},
			},
		},
		Metadata: map[string]interface{}{},
	}

	chatHistoryJSON, err := json.Marshal(chatHistory)
	if err != nil {
		return nil, fmt.Errorf("error marshaling chat history: %w", err)
	}

	// Create article
	article := &models.Article{
		Title:       title,
		Content:     finalMsg.Content[0].Text,
		Author:      authorID,
		IsDraft:     true,
		ChatHistory: chatHistoryJSON,
	}

	return article, nil
}

func (w *WriterAgent) UpdateWithContext(ctx context.Context, article *models.Article) (string, error) {
	if article == nil {
		return "", fmt.Errorf("article not found")
	}

	editorPrompt := `You are the Editor. Improve and refine the previously drafted content.
	Use the chat history to understand what the user wants and what the writer has written.`

	msg, err := w.client.Messages.New(ctx, anthropic.MessageNewParams{
		MaxTokens: 4096,
		Messages: []anthropic.MessageParam{
			{
				Role: anthropic.MessageParamRoleUser,
				Content: []anthropic.ContentBlockParamUnion{
					{
						OfRequestTextBlock: &anthropic.TextBlockParam{Text: editorPrompt},
					},
					{
						OfRequestTextBlock: &anthropic.TextBlockParam{Text: fmt.Sprintf("Title: %q\nPrompt: %s", article.Title, article.Content)},
					},
				},
			},
		},
		Model: anthropic.ModelClaude3_7SonnetLatest,
	})
	if err != nil {
		return "", fmt.Errorf("error updating article: %w", err)
	}

	return msg.Content[0].Text, nil
}
