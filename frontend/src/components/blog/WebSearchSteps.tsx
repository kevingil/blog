import {
  Source,
  SourceContent,
  SourceTrigger,
} from "@/components/prompt-kit/source"
import {
  Steps,
  StepsContent,
  StepsItem,
  StepsTrigger,
} from "@/components/prompt-kit/steps"

type SearchResult = {
  title: string;
  url: string;
  summary?: string;
  author?: string;
  published_date?: string;
  favicon?: string;
  highlights?: string[];
  text_preview?: string;
  has_full_text?: boolean;
};

type SourceInfo = {
  source_id: string;
  original_title: string;
  original_url: string;
  content_length: number;
  source_type?: string;
  search_query?: string;
};

interface WebSearchStepsProps {
  tool_context: {
    tool_name: string;
    tool_id: string;
    status: 'starting' | 'running' | 'completed' | 'error';
    search_query?: string;
    search_results?: SearchResult[];
    sources_created?: SourceInfo[];
    total_found?: number;
    sources_successful?: number;
    message?: string;
  };
}

export function WebSearchSteps({ tool_context }: WebSearchStepsProps) {
  const { status, search_query, search_results, sources_created, total_found, sources_successful, message } = tool_context;

  // Extract domain from URL for display
  const getDomain = (url: string) => {
    try {
      const urlObj = new URL(url);
      return urlObj.hostname.replace('www.', '');
    } catch {
      return url;
    }
  };

  return (
    <Steps defaultOpen={status === 'completed'}>
      <StepsTrigger>
        Web search: {search_query || 'researching...'}
      </StepsTrigger>
      
      <StepsContent>
        {/* Step 1: Searching */}
        <StepsItem status={status === 'starting' || status === 'running' ? 'running' : 'completed'}>
          Searching across curated sources...
        </StepsItem>
        
        {/* Step 2: Results found */}
        {total_found !== undefined && total_found > 0 && (
          <>
            <StepsItem status="completed">
              Found {total_found} {total_found === 1 ? 'match' : 'matches'}
            </StepsItem>
            
            {/* Display sources */}
            {search_results && search_results.length > 0 && (
              <div className="flex flex-wrap gap-1.5 mt-2">
                {search_results.slice(0, 6).map((result, idx) => (
                  <Source href={result.url} key={result.url || idx}>
                    <SourceTrigger 
                      label={getDomain(result.url)} 
                      showFavicon 
                    />
                    <SourceContent
                      title={result.title}
                      description={result.summary || result.text_preview || result.highlights?.[0]}
                      metadata={{
                        author: result.author,
                        published: result.published_date
                      }}
                    />
                  </Source>
                ))}
              </div>
            )}
          </>
        )}
        
        {/* Step 3: Creating sources */}
        {sources_created && sources_created.length > 0 && (
          <StepsItem status="completed">
            Created {sources_successful || sources_created.length} reference {sources_successful === 1 ? 'source' : 'sources'}
          </StepsItem>
        )}

        {/* Error state */}
        {status === 'error' && (
          <StepsItem status="error">
            {message || 'Search failed'}
          </StepsItem>
        )}

        {/* No results state */}
        {status === 'completed' && total_found === 0 && (
          <StepsItem status="completed">
            No results found for this query
          </StepsItem>
        )}
      </StepsContent>
    </Steps>
  );
}

