'use server'
import { ChatGroq } from "@langchain/groq";
import { BaseMessage, HumanMessage, SystemMessage } from "@langchain/core/messages";
import { StateGraph } from "@langchain/langgraph";
import { MemorySaver, Annotation, messagesStateReducer } from "@langchain/langgraph";
import { createArticle } from "@/components/blog/actions";
import { generateArticleImage } from "../images/generation";
import { DEFAULT_IMAGE_PROMPT } from "../images/const";
import { getArticles } from "@/components/blog/search";
// Store an array of messages in state
const StateAnnotation = Annotation.Root({
  messages: Annotation<BaseMessage[]>({
    reducer: messagesStateReducer,
  }),
});

const GROQ_KEY = process.env.GROQ_API_KEY;
const model = new ChatGroq({
  modelName: "llama-3.1-8b-instant",
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


  // Get writing context from the database
  const articles = await getArticles(1, null);
  const writingContext = articles.articles.map(article => 
    article?.content?.substring(0, 300).replace(/[#*`_]/g, '')
  ).join("\n\n");

  console.log("Writing Context:")
  console.log(writingContext)


  // Construct two calls:
  // First for the "originalWriter" node (system instructs to write a draft),
  // then for the "editor" node (system instructs to refine it).

  // Messages for the original writer
  const originalWriterSystem = new SystemMessage(
    `You are a ghostwriter. Draft a compelling blog article based on the given prompt for the author's provider title and prompt.
    This is what the author sounds like, these are some previous writings:
    ${writingContext}
    Please write a complete blog article with clear sections, engaging language, and relevant details.
    Please consider the author's voice and style.`
  );
  const userPrompt = new HumanMessage(
    `Title: "${title}"\nPrompt: ${prompt}`
  );

  // Messages for the editor
  const editorSystem = new SystemMessage(
    `You are the Editor. Improve and refine the previously drafted content.
    The blog should be formatted in markdown as follows: 

    [article intro - always start with this]
    ### Subheader
    [article body - there can be multiple subheaders and body sections depending on the article]
    ### Conclusion
    [article conclusion - always end with this]

    If there's code snippets, make sure to format in markdown.
    Link references can be added at the end of the article in markdown format.
    For listis, use unordered lists with -. 
    - Bullet point
    - Bullet point
    - Bullet point

    Writing style:
    Make it more concise, clear, and engaging but that it's following the author's voice and style
    Make sure the article makes sense and is coherent for a human and that it's not stating the obvious.
    Preserve the main idea and style, but ensure it's polished for publication.`
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
  const finalArticleContent = finalMessages[finalMessages.length - 1].content;

  console.log(finalState)

  // Create article from final content
  const newArticle = await createArticle({
    title,
    content: finalArticleContent.toString(),
    tags: [],
    isDraft: draft || false,
    authorId: authorId,
  });

  try {
    await generateArticleImage(DEFAULT_IMAGE_PROMPT[Math.floor(Math.random() * DEFAULT_IMAGE_PROMPT.length)], newArticle.id);
  } catch (error) {
    console.error("Error generating article image", error);
  }

  return newArticle;
}
