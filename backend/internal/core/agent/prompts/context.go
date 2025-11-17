// Package prompts provides all agent prompts for content generation
package prompts

// WritingContext contains example snippets from previous articles to maintain consistent tone and style.
// This helps the AI understand the author's voice and writing preferences.
const WritingContext = `
### How JavaScript Runs in MySQL
Oracle uses PL/SQL as the interface to run JavaScript on MySQL. You can define and save functions that you can later call in your queries. Although some versions of Oracle database already support JavaScript as stored procedures and inline with your query, MySQL only supports JavaScript as saved procedures for the time being. The code runs on the GraalVM runtime, which optimizes your code, converts it to machine code, then runs on the Graal JIT compiler.
### HTMX Frontend
Back on the homepage, we replace the template that was loading the articles with the code below. Using HTMX we easily implement lazy loading by displaying a placeholder as the initial state and calling the /chunks/feed endpoint that uses our new controller to load articles. Once we get a response, HTMX will handle the application state with hx-swap, in this case to replace the placeholder.
### First Day Hike
The hike on the first day did not take long, I started around noon, and finished at 4pm with several water, picture, and food breaks. The first lake is Carr Lake, where most day glampers go, I'm pretty sure I saw a TV setup. Next was, Feely Lake, and Milk Lake, where I stopped for Lunch.
### Running a Perl Script in a Dockerfile
One of the great things about Perl is that it ships with Linux out of the box. It's so well integrated with Unix, it can serve as a wrapper around system tools. Its strong support for text manipulation and data processing makes it very valuable when building distributed systems. When deploying complex Docker applications, there might be some pre-processing during the build process that can take advantage of Perl's many strong suits.
`

