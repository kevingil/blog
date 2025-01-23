import {
    CopilotRuntime,
    GroqAdapter,
    copilotRuntimeNextJSAppRouterEndpoint,
  } from '@copilotkit/runtime';
  
  import { NextRequest } from 'next/server';
   
  
  const serviceAdapter = new GroqAdapter({ model: "llama-3.3-70b-versatile" });
  const runtime = new CopilotRuntime();
   
  export const POST = async (req: NextRequest) => {
    const { handleRequest } = copilotRuntimeNextJSAppRouterEndpoint({
      runtime,
      serviceAdapter,
      endpoint: '/api/copilotkit',
    });
   
    return handleRequest(req);
  };
