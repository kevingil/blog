export type ArticleCitation = {
  id: string;
  title: string;
  url?: string;
};

export function ArticleSources({ sources }: { sources: ArticleCitation[] }) {
  const visible = sources.filter((source) => source.title.trim() || source.url?.trim());
  if (visible.length === 0) return null;

  return (
    <section className="article-sources" aria-label="Sources">
      <h2>Sources</h2>
      <ol>
        {visible.map((source) => {
          const label = source.title.trim() || source.url || 'Source';
          return (
            <li key={source.id}>
              {source.url ? (
                <a href={source.url} target="_blank" rel="noopener noreferrer">
                  {label}
                </a>
              ) : (
                label
              )}
            </li>
          );
        })}
      </ol>
    </section>
  );
}
