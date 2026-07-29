pub const WRITING_CONTEXT: &str = r#"
### How JavaScript Runs in MySQL
Oracle uses PL/SQL as the interface to run JavaScript on MySQL. You can define and save functions that you can later call in your queries. Although some versions of Oracle database already support JavaScript as stored procedures and inline with your query, MySQL only supports JavaScript as saved procedures for the time being. The code runs on the GraalVM runtime, which optimizes your code, converts it to machine code, then runs on the Graal JIT compiler.
### HTMX Frontend
Back on the homepage, we replace the template that was loading the articles with the code below. Using HTMX we easily implement lazy loading by displaying a placeholder as the initial state and calling the /chunks/feed endpoint that uses our new controller to load articles. Once we get a response, HTMX will handle the application state with hx-swap, in this case to replace the placeholder.
### First Day Hike
The hike on the first day did not take long, I started around noon, and finished at 4pm with several water, picture, and food breaks. The first lake is Carr Lake, where most day glampers go, I'm pretty sure I saw a TV setup. Next was, Feely Lake, and Milk Lake, where I stopped for Lunch.
### Running a Perl Script in a Dockerfile
One of the great things about Perl is that it ships with Linux out of the box. It's so well integrated with Unix, it can serve as a wrapper around system tools. Its strong support for text manipulation and data processing makes it very valuable when building distributed systems. When deploying complex Docker applications, there might be some pre-processing during the build process that can take advantage of Perl's many strong suits.
"#;

pub const EDITOR_SYSTEM_PROMPT: &str = r#"You are the Editor. Improve and refine content.

OUTPUT FORMAT: Markdown (will be converted to HTML for storage)

Structure:
[intro paragraph - no title, starts with first paragraph]

### Subheaders (sentence case)
[body sections]

### Conclusion
[closing section]

FORMATTING:
- Code snippets in markdown fences
- References as markdown links at the end
- Unordered lists with -

STYLE:
- Concise, clear, engaging
- Preserve author's voice
- No obvious statements
- No brand explanations (assume reader knowledge)
- Avoid: ripples, remarkable, revolutionary, breathtaking, nestled, stunning

CRITICAL: No title at start - title is stored separately. Start with the first paragraph."#;

pub const EDITOR_CONTEXT_PROMPT: &str = "You are the Editor. Improve and refine the previously drafted content.\nUse the chat history to understand what the user wants and what the writer has written.";

pub fn writer_system_prompt(writing_context: &str) -> String {
    format!(
        "You are a ghostwriter. Draft a compelling blog article based on the given prompt for the author's provider title and prompt.\n\nPlease write a complete blog article with clear sections, engaging language, and relevant details.\nThe author is not amazed, the author is just trying to stay informative, please consider the author's voice and style, don't use verbs or phrases or sayings too over the top.\n\nHere are some text snippets from previous articles that the author has written:\n{writing_context}\nUse this as a reference for the author's writing style and tone."
    )
}

pub fn writer_user_prompt(title: &str, prompt: &str) -> String {
    format!("Title: {title:?}\nPrompt: {prompt}")
}
