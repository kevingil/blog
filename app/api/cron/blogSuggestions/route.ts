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
      modelName: "llama-3.1-8b-instant",
      temperature: 0.7,
      apiKey: process.env.GROQ_API_KEY,
    });

    // Get writing context from recent articles
    const recentArticleTitles = await db.select({ title: articles.title })
      .from(articles)
      .where(eq(articles.isDraft, false))
      .orderBy(desc(articles.createdAt))
      .limit(5)
      .execute();

    const writingContext = recentArticleTitles.map(article => article.title).join("\n");

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
      These are prvious writings from the author: ${writingContext}
      And these latest trends and news: ${JSON.stringify(searchResults.results.slice(0, 3))}
      Don't suggest the same ideas as previous articles, suggest something new and engaging.
      Make the article informative and engaging.`);


    const articleIdea = await structuredResponse.invoke([
      new SystemMessage(`Based on the following context and search results, suggest an engaging article title about technology or AI:
        These are prvious writings from the author: ${writingContext}
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
