import { describe, expect, test } from 'bun:test';
import { Marked } from 'marked';
import { addedTextRanges } from '../../../lib/added-text';
import { htmlToMarkdown, markdownToHtml } from './markdown';

describe('markdownToHtml highlights', () => {
  test('highlights an inserted paragraph and leaves the rest alone', () => {
    const before = 'Intro\n\nEnding';
    const after = 'Intro\n\nA new paragraph.\n\nEnding';
    const html = markdownToHtml(after, addedTextRanges(before, after));
    expect(html).toContain('<p>Intro</p>');
    expect(html).toContain('<p><mark data-added="true">A new paragraph.</mark></p>');
    expect(html).toContain('<p>Ending</p>');
    expect(html.match(/<mark /g)).toHaveLength(1);
  });

  test('highlights a full replacement', () => {
    const html = markdownToHtml('completely new', addedTextRanges('old draft', 'completely new'));
    expect(html).toContain('<mark data-added="true">completely new</mark>');
  });

  test('leaves deletions unmarked', () => {
    const html = markdownToHtml('keep\n\nstay', addedTextRanges('keep\n\nremove\n\nstay', 'keep\n\nstay'));
    expect(html).not.toContain('data-added');
    expect(markdownToHtml('same', addedTextRanges('same', 'same'))).not.toContain('data-added');
  });

  test('highlights only a newly added list item', () => {
    const before = '- a\n- b\n';
    const after = '- a\n- b\n- c\n';
    const html = markdownToHtml(after, addedTextRanges(before, after));
    expect(html).toContain('<mark data-added="true">c</mark>');
    expect(html).not.toMatch(/<mark[^>]*>a</);
    expect(html).not.toMatch(/<mark[^>]*>b</);
    expect(html).toContain('<li><p>a</p></li>');
  });

  test('highlights inline formatting inside a new paragraph', () => {
    const html = markdownToHtml('Hello **world**', [{ from: 0, to: 20 }]);
    expect(html).toContain('<mark data-added="true">Hello </mark>');
    expect(html).toContain('<strong><mark data-added="true">world</mark></strong>');
  });

  test('marks a new code block', () => {
    const before = 'Intro';
    const after = 'Intro\n\n```js\nconst x = 1;\n```\n';
    const html = markdownToHtml(after, addedTextRanges(before, after));
    expect(html).toContain('<pre data-added="true">');
    expect(html).toContain('language-js');
    expect(html).toContain('<mark data-added="true">const x = 1;</mark>');
    expect(html).not.toMatch(/<mark[^>]*>Intro/);
  });

  test('lexer tokens cover the source', () => {
    const markdown = '## Title\n\nHello **bold**\n\n- a\n- b\n\n> quote\n';
    const tokens = new Marked().lexer(markdown);
    const covered = tokens.reduce((sum, token) => sum + (token.raw?.length ?? 0), 0);
    expect(covered).toBe(markdown.length);
  });
});

describe('htmlToMarkdown', () => {
  test('strips highlight marks', () => {
    const markdown = htmlToMarkdown('<p><mark data-added="true">Hello </mark><strong><mark data-added="true">bold</mark></strong></p>');
    expect(markdown).toBe('Hello **bold**');
    expect(markdown).not.toContain('mark');
  });

  test('keeps headings, links, images, and code', () => {
    const markdown = htmlToMarkdown([
      '<h2>Title</h2>',
      '<p>Hello <strong>bold</strong> and <em>em</em> and <code>code</code>.</p>',
      '<pre><code class="language-js">const x = 1;</code></pre>',
      '<p><a href="https://example.com">link</a></p>',
      '<p><img src="https://example.com/a.png" alt="alt" data-added="true"></p>',
    ].join(''));
    expect(markdown).toContain('## Title');
    expect(markdown).toContain('**bold**');
    expect(markdown).toContain('*em*');
    expect(markdown).toContain('`code`');
    expect(markdown).toContain('```js\nconst x = 1;\n```');
    expect(markdown).toContain('[link](https://example.com)');
    expect(markdown).toContain('![alt](https://example.com/a.png)');
    expect(markdown).not.toContain('data-added');
  });

  test('writes tight lists from tiptap paragraphs', () => {
    const markdown = htmlToMarkdown('<ul><li><p>a</p></li><li><p>b</p></li></ul>');
    expect(markdown).toBe('- a\n- b');
  });
});
