import { Agent } from "@/client";
import type {
  ConversationTurnRequest,
  ConversationTurnResponse,
} from "@/client";
import { generatedData } from "./generatedClient";

export async function submitConversationTurn(
  request: ConversationTurnRequest,
): Promise<ConversationTurnResponse> {
  return generatedData<ConversationTurnResponse>(
    Agent.submitConversationTurn({ body: request }),
  );
}
