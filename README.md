# Blog Copilot

An agentic blog editor.

![frontend/public/screenshot-image.png](frontend/public/screenshot-image.png)


![frontend/public/Xnip2025-06-29_15-58-49.jpg](frontend/public/Xnip2025-06-29_15-58-49.jpg)


## Getting Started

### Frontend Setup

Navigate to the frontend directory and install dependencies:
```bash
cd frontend
bun install
```

Start the development server:
```bash
bun run dev
```

Build for production:
```bash
bun run build
```

### Backend Setup (Go)

The Makefile commands are for the backend only:

Run build make command with tests:
```bash
make all
```

Build the application:
```bash
make build
```

Run the application:
```bash
make run
```

Live reload the application:
```bash
make watch
```

Run the test suite:
```bash
make test
```

Clean up binary from the last build:
```bash
make clean
```

### API Client Generation

The frontend TypeScript client is auto-generated from the backend's OpenAPI spec using [@hey-api/openapi-ts](https://heyapi.dev/).

Regenerate the client after API changes:
```bash
make generate-client
```

This runs `swag init` to generate Swagger docs from Go annotations, then generates typed TypeScript SDK classes in `frontend/src/client/`.

**Usage:**
```typescript
import { Articles, Auth } from './client'

const { data } = await Auth.postAuthLogin({ body: { email, password } })
const articles = await Articles.getBlogArticles({ query: { page: 1 } })
```

### Quick Overview


#### Backend clean code architecture layers


Dependencies point inward toward core.

```mermaid
flowchart TB
    subgraph outer [External]
        HTTP[HTTP/Fiber]
        DB[PostgreSQL/GORM]
        External[OpenAI/Exa/S3]
    end
    
    subgraph layers [Internal Layers]
        Handlers
        Repository[Repository - Interfaces + Implementations]
        Integrations
    end
    
    subgraph core [Core - Business Logic]
        Types[Types]
        Services[Services]
        Errors[Errors]
    end
    
    HTTP --> Handlers
    Handlers --> Services
    Services --> Types
    Services --> Repository
    Repository --> DB
    Integrations --> External
```



#### Data Pipeline and Insights System


```mermaid
flowchart TB
    subgraph userLayer [User Layer]
        direction LR
        UI_DS[Data Sources Settings]
        UI_INS[Insights Page]
        UI_EDIT[Article Editor]
    end

    subgraph apiLayer [API Layer]
        direction LR
        API_DS[/data-sources API/]
        API_INS[/insights API/]
        API_AGENT[/agent API/]
    end

    subgraph serviceLayer [Service Layer]
        direction LR
        SVC_DS[DataSourceService]
        SVC_INS[InsightService]
        SVC_CRAWL[CrawledContentService]
    end

    subgraph workerLayer [Background Workers]
        direction TB
        SCHED[Scheduler - Cron]
        
        subgraph workers [Worker Pool]
            CW[CrawlWorker]
            IW[InsightWorker]
            DW[DiscoveryWorker]
        end
        
        SCHED --> CW
        SCHED --> IW
        SCHED --> DW
    end

    subgraph externalServices [External Services]
        direction LR
        EXA[Exa Search API]
        OPENAI[OpenAI Embeddings]
        LLM[LLM Provider]
    end

    subgraph dataLayer [Database - PostgreSQL + pgvector]
        direction TB
        
        subgraph tables [Tables]
            T_DS[(data_source)]
            T_CC[(crawled_content)]
            T_IT[(insight_topic)]
            T_INS[(insight)]
            T_CTM[(content_topic_match)]
        end
        
        subgraph vectors [Vector Indexes - IVFFlat]
            V_CC[content_embedding_idx]
            V_IT[topic_embedding_idx]
            V_INS[insight_embedding_idx]
        end
        
        T_CC --- V_CC
        T_IT --- V_IT
        T_INS --- V_INS
    end

    subgraph agentTools [Agent Tools]
        direction LR
        TOOL_INS[get_insights]
        TOOL_SEARCH[search_crawled_content]
        TOOL_SUM[summarize_sources]
    end

    UI_DS --> API_DS
    UI_INS --> API_INS
    UI_EDIT --> API_AGENT

    API_DS --> SVC_DS
    API_INS --> SVC_INS
    API_AGENT --> agentTools

    SVC_DS --> T_DS
    SVC_INS --> T_INS
    SVC_INS --> T_IT
    SVC_CRAWL --> T_CC

    CW -->|fetch content| T_DS
    CW -->|store content| T_CC
    CW -->|generate| OPENAI

    IW -->|read content| T_CC
    IW -->|match topics via embedding| T_IT
    IW -->|generate summary| LLM
    IW -->|store insight| T_INS

    DW -->|find similar sites| EXA
    DW -->|create discovered| T_DS

    agentTools -->|semantic search| V_CC
    agentTools -->|semantic search| V_INS
    agentTools -->|topic matching| V_IT
```
