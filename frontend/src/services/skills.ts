import { Agent } from "@/client";
import type {
  SkillListResponse,
  SkillResponse,
  SkillUpdateRequest,
  SkillWriteRequest,
} from "@/client";
import { generatedData } from "./generatedClient";

export async function listAgentSkills(): Promise<SkillListResponse> {
  return generatedData<SkillListResponse>(Agent.listAgentSkills());
}

export async function createAgentSkill(
  request: SkillWriteRequest,
): Promise<SkillResponse> {
  return generatedData<SkillResponse>(Agent.createAgentSkill({ body: request }));
}

export async function updateAgentSkill(
  skillId: string,
  request: SkillUpdateRequest,
): Promise<SkillResponse> {
  return generatedData<SkillResponse>(
    Agent.updateAgentSkill({
      path: { skillId },
      body: request,
    }),
  );
}

export async function deleteAgentSkill(
  skillId: string,
): Promise<{ success: boolean }> {
  return generatedData<{ success: boolean }>(
    Agent.deleteAgentSkill({ path: { skillId } }),
  );
}
