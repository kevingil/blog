'use server'
import { NextResponse } from 'next/server';
import { tavily } from "@tavily/core";
import { ChatGroq } from "@langchain/groq";
import { SystemMessage } from "@langchain/core/messages";
import { articles } from '@/db/schema';
import { db } from '@/db/drizzle';
import { eq, sql, desc } from 'drizzle-orm';
import { getArticles } from '@/components/blog/search';
import { generateArticle } from '@/lib/llm/articles';
import { NextRequest } from 'next/server';
import { z } from 'zod';
import { zodToJsonSchema } from 'zod-to-json-schema';


const articleIdeaSchema = z.object({
  title: z.string(),
  prompt: z.array(z.object({
    articleIdea: z.string(),
    webReferences: z.string()
  }))
});


export async function GET(req: NextRequest) {
  
  if (req.headers.get('Authorization') !== `Bearer ${process.env.CRON_SECRET}`) {
    console.log('Unauthorized');
    return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
  }

  try {
    // Check current draft count
    console.log('Checking draft count');
    const draftCounts = await db.select({ count: sql<number>`count(*)` })
    .from(articles)
    .where(eq(articles.isDraft, true))
    .execute();

    console.log('Draft count:', draftCounts[0].count);
    if (draftCounts[0].count >= 20) {
      return NextResponse.json({ message: "Sufficient drafts exist" });
    }

   
    console.log('Initializing Groq model');
    const model = new ChatGroq({
      modelName: "deepseek-r1-distill-llama-70b",
      temperature: 0.7,
      apiKey: process.env.GROQ_API_KEY,
    });

    // // Get writing context from recent articles
    // const recentArticleTitles = await db.select({ title: articles.title })
    //   .from(articles)
    //   .where(eq(articles.isDraft, false))
    //   .orderBy(desc(articles.createdAt))
    //   .limit(5)
    //   .execute();

    // const writingContext = recentArticleTitles.map(article => article.title).join("\n");
    const writingContext = `
Here are some text snippets from previous articles that the author has written:

### How JavaScript Runs in MySQL
Oracle uses PL/SQL as the interface to run JavaScript on MySQL. You can define and save functions that you can later call in your queries. Although some versions of Oracle database already support JavaScript as stored procedures and inline with your query, MySQL only supports JavaScript as saved procedures for the time being. The code runs on the GraalVM runtime, which optimizes your code, converts it to machine code, then runs on the Graal JIT compiler.
### HTMX Frontend
Back on the homepage, we replace the template that was loading the articles with the code below. Using HTMX we easily implement lazy loading by displaying a placeholder as the initial state and calling the /chunks/feed endpoint that uses our new controller to load articles. Once we get a response, HTMX will handle the application state with hx-swap, in this case to replace the placeholder.
### First Day Hike
The hike on the first day did not take long, I started around noon, and finished at 4pm with several water, picture, and food breaks. The first lake is Carr Lake, where most day glampers go, I'm pretty sure I saw a TV setup. Next was, Feely Lake, and Milk Lake, where I stopped for Lunch.
### Running a Perl Script in a Dockerfile
One of the great things about Perl is that it ships with Linux out of the box. It's so well integrated with Unix, it can serve as a wrapper around system tools. Its strong support for text manipulation and data processing makes it very valuable when building distributed systems. When deploying complex Docker applications, there might be some pre-processing during the build process that can take advantage of Perl's many strong suits.

Use this as a reference for the author's writing style and tone.
    `;

     // Initialize clients
     console.log('Initializing Tavily client');
     const tavilyClient = tavily({ apiKey: process.env.TAVILY_API_KEY });

    // Get latest news/trends from Tavily
    console.log('Searching for latest news/trends');
    const searchResults = await tavilyClient.search(
      `Search the latest on software engineering, AI news, agents, machine learning, langchain, technical papers, and trends. 
      Good sources include hacker news (news.ycombinator.com), linkedin, or technical blogs from tech leaders. `, 
      {
        searchDepth: "advanced"
      }
    );

    console.log('Search results:', searchResults);

    const structuredResponse = await model.withStructuredOutput(articleIdeaSchema)

    console.log(`Based on the following context and search results, suggest an engaging article title about technology or AI:
      And these latest trends and news: ${JSON.stringify(searchResults.results.slice(0, 3))}
      Don't suggest the same ideas as previous articles, suggest something new and engaging.
      Make the article informative and engaging.`);


    const articleIdea = await structuredResponse.invoke([
      new SystemMessage(`Based on the following context and search results, suggest an engaging article title about technology or AI:
        And these latest trends and news: ${JSON.stringify(searchResults.results.slice(0, 3))}
        Don't suggest the same ideas as previous articles, suggest something new and engaging.
        Make the article informative and engaging.`)
    ]);

    


    console.log('Suggested prompt:', articleIdea.prompt);

    const formattedPrompt = articleIdea.prompt.map(item => item.articleIdea).join('\n');

    // Generate and save the article
    await generateArticle(formattedPrompt, articleIdea.title as string, 1, true);

    return NextResponse.json({ 
      success: true, 
      message: "Article draft generated",
      title: articleIdea.title,
    });

  } catch (error) {
    console.error('Error in cron job:', error);
    return NextResponse.json({ error: 'Failed to generate article' }, { status: 500 });
  }
}

// // Add configuration for cron schedule
// export const config = {
//   runtime: 'edge',
//   schedule: '0 */6 * * *' // Runs every 6 hours
// };
