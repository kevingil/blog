'use server'
import { ChatGroq } from "@langchain/groq";
import { BaseMessage, HumanMessage, SystemMessage } from "@langchain/core/messages";
import { StateGraph } from "@langchain/langgraph";
import { MemorySaver, Annotation, messagesStateReducer } from "@langchain/langgraph";
import { createArticle } from "@/components/blog/actions";
import { generateArticleImage } from "../images/generation";
import { DEFAULT_IMAGE_PROMPT } from "../images/const";
import { getArticle } from "@/components/blog/search";


import { eq, sql, or, like, and, inArray, desc } from 'drizzle-orm';
import { db } from '@/db/drizzle';
import { articles, users, articleTags, tags } from '@/db/schema';
import { ArticleListItem, ITEMS_PER_PAGE } from '@/components/blog/index';
import { log } from 'console';


export interface ArticleChatHistoryMessage {
  role: "user" | "assistant" | "system";
  content: string;
  created_at: number;
  metadata: any;
}

export interface ArticleChatHistory {
  messages: ArticleChatHistoryMessage[];
  metadata: any;
}


// Store an array of messages in state
const StateAnnotation = Annotation.Root({
  messages: Annotation<BaseMessage[]>({
    reducer: messagesStateReducer,
  }),
});

const GROQ_KEY = process.env.GROQ_API_KEY;
const model = new ChatGroq({
  modelName: "deepseek-r1-distill-llama-70b",
  temperature: 0.7,
  apiKey: GROQ_KEY,
});

// Langraph Notes:
// For groq, we need to specify the message in the array of messages in the graph state
// For openai, we can just pass the messages array

// Original writer node
async function originalWriterCall(state: typeof StateAnnotation.State) {
  const messages = state.messages.slice(0, 2); // Only take system and user messages for writer
  const response = await model.invoke(messages);
  return { messages: [response] };
}

// Editor node
async function editorCall(state: typeof StateAnnotation.State) {
  const previousContent = state.messages[state.messages.length - 1].content;
  const editorMessages = [
    state.messages[2], // editor system message
    new HumanMessage(previousContent.toString()) // previous content as input
  ];
  const response = await model.invoke(editorMessages);
  return { messages: [response] };
}

// Workflow chain
const workflow = new StateGraph(StateAnnotation)
  .addNode("originalWriter", originalWriterCall)
  .addNode("editor", editorCall)
  .addEdge("__start__", "originalWriter")
  .addEdge("originalWriter", "editor")
  .addEdge("editor", "__end__");

const checkpointer = new MemorySaver();
const app = workflow.compile({ checkpointer });

export async function generateArticle(prompt: string, title: string, authorId: number, draft?: boolean) {


  // Get writing context, handpicked
  const writingContext = `
### How JavaScript Runs in MySQL
Oracle uses PL/SQL as the interface to run JavaScript on MySQL. You can define and save functions that you can later call in your queries. Although some versions of Oracle database already support JavaScript as stored procedures and inline with your query, MySQL only supports JavaScript as saved procedures for the time being. The code runs on the GraalVM runtime, which optimizes your code, converts it to machine code, then runs on the Graal JIT compiler.
### HTMX Frontend
Back on the homepage, we replace the template that was loading the articles with the code below. Using HTMX we easily implement lazy loading by displaying a placeholder as the initial state and calling the /chunks/feed endpoint that uses our new controller to load articles. Once we get a response, HTMX will handle the application state with hx-swap, in this case to replace the placeholder.
### First Day Hike
The hike on the first day did not take long, I started around noon, and finished at 4pm with several water, picture, and food breaks. The first lake is Carr Lake, where most day glampers go, I'm pretty sure I saw a TV setup. Next was, Feely Lake, and Milk Lake, where I stopped for Lunch.
### Running a Perl Script in a Dockerfile
One of the great things about Perl is that it ships with Linux out of the box. It's so well integrated with Unix, it can serve as a wrapper around system tools. Its strong support for text manipulation and data processing makes it very valuable when building distributed systems. When deploying complex Docker applications, there might be some pre-processing during the build process that can take advantage of Perl's many strong suits.
    `;


  // Construct two calls:
  // First for the "originalWriter" node (system instructs to write a draft),
  // then for the "editor" node (system instructs to refine it).

  // Messages for the original writer
  const originalWriterSystem = new SystemMessage(
    `You are a ghostwriter. Draft a compelling blog article based on the given prompt for the author's provider title and prompt.

    Please write a complete blog article with clear sections, engaging language, and relevant details.
    The author is not amazed, the author is just trying to stay informative, please consider the author's voice and style, don't use verbs or phrases or sayings too over the top.

    Here are some text snippets from previous articles that the author has written:
    ${writingContext}
    Use this as a reference for the author's writing style and tone.

    These are words and phrases to avoid, try not using these words:




    `
  );
  const userPrompt = new HumanMessage(
    `Title: "${title}"\nPrompt: ${prompt}`
  );

  // Messages for the editor
  const editorSystem = new SystemMessage(
    `You are the Editor. Improve and refine the previously drafted content.
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

    Here are some text snippets from previous articles that the author has written:
    ${writingContext}
    Use this as a reference for the author's writing style and tone.
    
    IMPORTANT: 
    the article should not start with a title or subheader, the title is saved somewhere else, we just want the content, start with the first paragraph of the article.
    The voice is the most important, make sure you don't use worlds like ripples or remarkable, just use the words that the author would use.
    `
  );
  // Run the workflow
  const finalState = await app.invoke(
    {
      messages: [
        originalWriterSystem,
        userPrompt,
        editorSystem,
      ],
    },
    // Thread ID to store conversation in memory
    { configurable: { thread_id: "article_generation_thread" } }
  );

  // Final message from the editor node
  const finalMessages = finalState.messages;
  let finalArticleContent = finalMessages[finalMessages.length - 1].content;

  console.log(finalState)
  finalArticleContent = finalArticleContent.toString().replace(/<think>[\s\S]*?<\/think>/g, '').trim();
  console.log(finalArticleContent)

  // Create article from final content
  const newArticle = await createArticle({
    title,
    content: finalArticleContent,
    tags: [],
    isDraft: true,
    authorId: authorId,
  });

  try {
    await generateArticleImage(DEFAULT_IMAGE_PROMPT[Math.floor(Math.random() * DEFAULT_IMAGE_PROMPT.length)], newArticle.id);
  } catch (error) {
    console.error("Error generating article image", error);
  }

  return newArticle;
}


export async function getArticleChatHistory(articleId: number): Promise<ArticleChatHistory | null> {

  // Fetch chat history stored as string and asssign a type

  const [row] = await db
    .select({ chatHistory: articles.chatHistory })
    .from(articles)
    .where(eq(articles.id, articleId))
    .limit(1);
  
  if (row && row.chatHistory) {
    return JSON.parse(row.chatHistory) as ArticleChatHistory;
  }
  return null;
}


export async function updateWithContext(articleId: number): Promise<{ content: string, success: boolean } | null> {

  const article = await getArticle(articleId);

  if (!article) {
    console.error("Article not found");
    return null;
  }

  //const articleChatHistory = await getArticleChatHistory(articleId);

  // Editor prompt 
  const editorPrompt = new SystemMessage(
    `You are the Editor. Improve and refine the previously drafted content.
    Use the chat history to understahd what the user wants and what the writer has written.
    `
  );

  const userPrompt = new HumanMessage(
    `Title: "${article.title}"\nPrompt: ${article.content}
    Improve the article based on user needs, see notes left by the writer in [notes: ... ].
    `
  );

  const messages = [
    editorPrompt,
    userPrompt,
  ];

  const response = await model.invoke(messages);


  console.log(response.content)

  return {
    content: response.content.toString().replace(/<think>[\s\S]*?<\/think>/g, '').trim(),
    success: true,
  };

}
