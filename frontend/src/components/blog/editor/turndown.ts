import TurndownService from 'turndown';

// #region agent log
console.log('[TURNDOWN MODULE] Loading turndown.ts module');
fetch('http://127.0.0.1:7242/ingest/5ed2ef34-0520-4861-bbfe-52c16271e660',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({location:'turndown.ts:module_load',message:'turndown module loaded',data:{},timestamp:Date.now(),hypothesisId:'H7'})}).catch(()=>{});
// #endregion

// Module-level singleton: properly configured Turndown instance.
const turndownService = new TurndownService({
  headingStyle: 'atx',
  codeBlockStyle: 'fenced',
  emDelimiter: '*',
  bulletListMarker: '-',
});

// Custom rule: TipTap's <pre><code> blocks must produce proper fenced code blocks.
// Without this, Turndown escapes backticks/special chars and flattens code to one line.
turndownService.addRule('fencedCodeBlock', {
  filter: function (node: HTMLElement) {
    const matches = node.nodeName === 'PRE';
    // #region agent log
    if (matches) {
      console.log('[TURNDOWN] fencedCodeBlock rule MATCHED on PRE element', node.innerHTML?.substring(0, 100));
      fetch('http://127.0.0.1:7242/ingest/5ed2ef34-0520-4861-bbfe-52c16271e660',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({location:'turndown.ts:filter',message:'PRE filter matched',data:{innerHTML:(node.innerHTML||'').substring(0,200)},timestamp:Date.now(),hypothesisId:'H7'})}).catch(()=>{});
    }
    // #endregion
    return matches;
  },
  replacement: function (_content: string, node: Node) {
    const el = node as HTMLElement;
    const codeEl = el.querySelector('code');
    const text = (codeEl || el).textContent || '';
    const langClass = codeEl?.getAttribute('class') || '';
    const langMatch = langClass.match(/language-(\S+)/);
    const lang = langMatch ? langMatch[1] : '';
    const result = '\n\n```' + lang + '\n' + text + '\n```\n\n';
    // #region agent log
    console.log('[TURNDOWN] fencedCodeBlock replacement called, lang:', lang, 'textLen:', text.length);
    fetch('http://127.0.0.1:7242/ingest/5ed2ef34-0520-4861-bbfe-52c16271e660',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({location:'turndown.ts:replacement',message:'PRE replacement called',data:{lang,textLen:text.length,resultFirst100:result.substring(0,100)},timestamp:Date.now(),hypothesisId:'H7'})}).catch(()=>{});
    // #endregion
    return result;
  },
});

export { turndownService };
