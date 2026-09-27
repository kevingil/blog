import { Agent } from "@/client";
import type {
  AgentToolListResponse,
  ConnectorListResponse,
  ConnectorPresetListResponse,
  ConnectorRefreshResponse,
  ConnectorResponse,
  ConnectorUpdateRequest,
  ConnectorWriteRequest,
  OauthConnectResponse,
  OauthConnectorRequest,
} from "@/client";
import { generatedData } from "./generatedClient";

export async function listAgentTools(): Promise<AgentToolListResponse> {
  return generatedData<AgentToolListResponse>(Agent.listAgentTools());
}

export async function listMcpConnectors(): Promise<ConnectorListResponse> {
  return generatedData<ConnectorListResponse>(Agent.listMcpConnectors());
}

export async function createMcpConnector(
  request: ConnectorWriteRequest,
): Promise<ConnectorResponse> {
  return generatedData<ConnectorResponse>(
    Agent.createMcpConnector({ body: request }),
  );
}

export async function updateMcpConnector(
  connectorId: string,
  request: ConnectorUpdateRequest,
): Promise<ConnectorResponse> {
  return generatedData<ConnectorResponse>(
    Agent.updateMcpConnector({
      path: { connectorId },
      body: request,
    }),
  );
}

export async function deleteMcpConnector(
  connectorId: string,
): Promise<{ success: boolean }> {
  return generatedData<{ success: boolean }>(
    Agent.deleteMcpConnector({ path: { connectorId } }),
  );
}

export async function refreshMcpConnector(
  connectorId: string,
): Promise<ConnectorRefreshResponse> {
  return generatedData<ConnectorRefreshResponse>(
    Agent.refreshMcpConnector({ path: { connectorId } }),
  );
}

export async function listConnectorPresets(): Promise<ConnectorPresetListResponse> {
  return generatedData<ConnectorPresetListResponse>(Agent.listConnectorPresets());
}

export async function connectConnectorPreset(
  presetId: string,
): Promise<OauthConnectResponse> {
  return generatedData<OauthConnectResponse>(
    Agent.connectConnectorPreset({ path: { presetId } }),
  );
}

export async function connectOauthMcp(
  request: OauthConnectorRequest,
): Promise<OauthConnectResponse> {
  return generatedData<OauthConnectResponse>(
    Agent.connectOauthMcp({ body: request }),
  );
}
