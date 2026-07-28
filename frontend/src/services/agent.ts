import { Agent } from "@/client";
import type { ChatRequest, ChatRequestResponse } from "@/client";
import { generatedData } from "./generatedClient";

export async function submitAgentRequest(
  request: ChatRequest,
): Promise<ChatRequestResponse> {
  return generatedData<ChatRequestResponse>(
    Agent.submitAgentRequest({ body: request }),
  );
}
