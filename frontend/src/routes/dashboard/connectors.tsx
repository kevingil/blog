import { useEffect, useState } from "react";
import { createFileRoute } from "@tanstack/react-router";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Loader2, Plug, Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useToast } from "@/hooks/use-toast";
import { useAdminDashboard } from "@/services/dashboard/dashboard";
import {
  connectConnectorPreset,
  connectOauthMcp,
  deleteMcpConnector,
  listConnectorPresets,
  listMcpConnectors,
} from "@/services/connectors";

export const Route = createFileRoute("/dashboard/connectors")({
  component: ConnectorsPage,
});

function ConnectorsPage() {
  const { setPageTitle } = useAdminDashboard();
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const [name, setName] = useState("");
  const [url, setUrl] = useState("");
  const [pendingId, setPendingId] = useState("");

  useEffect(() => {
    setPageTitle("Connectors");
  }, [setPageTitle]);

  const presetsQuery = useQuery({
    queryKey: ["connector-presets"],
    queryFn: listConnectorPresets,
  });
  const connectorsQuery = useQuery({
    queryKey: ["mcp-connectors"],
    queryFn: listMcpConnectors,
  });

  const invalidate = async () => {
    await queryClient.invalidateQueries({ queryKey: ["connector-presets"] });
    await queryClient.invalidateQueries({ queryKey: ["mcp-connectors"] });
    await queryClient.invalidateQueries({ queryKey: ["agent-tools"] });
  };

  const finishConnect = async (
    label: string,
    result: { connected: boolean; authorizationUrl: string },
  ) => {
    if (result.connected) {
      await invalidate();
      toast({ title: `${label} connected` });
      return;
    }
    if (result.authorizationUrl) {
      window.open(result.authorizationUrl, "_blank", "noopener,noreferrer");
      toast({
        title: `Sign in to ${label}`,
        description: "Finish sign-in with the provider, then click Connect again.",
      });
    }
  };

  const presetIds = new Set(
    (presetsQuery.data?.presets ?? [])
      .map((preset) => preset.connectorId)
      .filter((id) => id.length > 0),
  );
  const customConnectors = (connectorsQuery.data?.connectors ?? []).filter(
    (connector) => !presetIds.has(connector.id) && !connector.env.preset_id,
  );

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-6 p-6">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Connectors</h1>
        <p className="text-sm text-muted-foreground">
          Sources of context for the agent. Connect a preset with one click, or add any other OAuth MCP server.
        </p>
      </div>

      <div className="grid gap-3">
        {(presetsQuery.data?.presets ?? []).map((preset) => {
          const busy = pendingId === preset.id;
          return (
            <Card key={preset.id}>
              <CardHeader className="flex flex-row items-start justify-between gap-4 space-y-0">
                <div className="space-y-1">
                  <CardTitle className="flex items-center gap-2 text-base">
                    <Plug className="h-4 w-4" />
                    {preset.name}
                  </CardTitle>
                  <CardDescription>{preset.description}</CardDescription>
                  <p className="text-xs text-muted-foreground">{preset.officialUrl}</p>
                  {preset.lastError && (
                    <p className="text-xs text-destructive">{preset.lastError}</p>
                  )}
                </div>
                {preset.connected ? (
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={busy}
                    onClick={async () => {
                      setPendingId(preset.id);
                      try {
                        await deleteMcpConnector(preset.connectorId);
                        await invalidate();
                        toast({ title: `${preset.name} disconnected` });
                      } catch (error) {
                        toast({
                          title: "Could not disconnect",
                          description: error instanceof Error ? error.message : "Try again",
                          variant: "destructive",
                        });
                      } finally {
                        setPendingId("");
                      }
                    }}
                  >
                    {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : "Disconnect"}
                  </Button>
                ) : (
                  <Button
                    size="sm"
                    disabled={busy}
                    onClick={async () => {
                      setPendingId(preset.id);
                      try {
                        const result = await connectConnectorPreset(preset.id);
                        await finishConnect(preset.name, result);
                      } catch (error) {
                        toast({
                          title: "Could not connect",
                          description: error instanceof Error ? error.message : "Try again",
                          variant: "destructive",
                        });
                      } finally {
                        setPendingId("");
                      }
                    }}
                  >
                    {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : "Connect"}
                  </Button>
                )}
              </CardHeader>
            </Card>
          );
        })}
        {presetsQuery.isLoading && (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 className="h-4 w-4 animate-spin" />
            Loading connectors
          </div>
        )}
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Another OAuth MCP</CardTitle>
          <CardDescription>
            Add a server by name and URL. Connect uses the same sign-in path as the presets.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="grid gap-2">
            <Label htmlFor="oauth-mcp-name">Name</Label>
            <Input
              id="oauth-mcp-name"
              value={name}
              onChange={(event) => setName(event.target.value)}
              placeholder="Team wiki"
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="oauth-mcp-url">MCP server URL</Label>
            <Input
              id="oauth-mcp-url"
              value={url}
              onChange={(event) => setUrl(event.target.value)}
              placeholder="https://example.com/mcp"
            />
          </div>
          <Button
            size="sm"
            disabled={!name.trim() || !url.trim() || pendingId === "custom"}
            onClick={async () => {
              const label = name.trim();
              setPendingId("custom");
              try {
                const result = await connectOauthMcp({ name: label, url: url.trim() });
                if (result.connected) {
                  setName("");
                  setUrl("");
                }
                await finishConnect(label, result);
              } catch (error) {
                toast({
                  title: "Could not connect",
                  description: error instanceof Error ? error.message : "Try again",
                  variant: "destructive",
                });
              } finally {
                setPendingId("");
              }
            }}
          >
            {pendingId === "custom" ? (
              <Loader2 className="mr-1 h-4 w-4 animate-spin" />
            ) : (
              <Plus className="mr-1 h-4 w-4" />
            )}
            Connect
          </Button>
        </CardContent>
      </Card>

      {customConnectors.length > 0 && (
        <div className="space-y-2">
          <h2 className="text-sm font-medium">Other connected servers</h2>
          {customConnectors.map((connector) => (
            <Card key={connector.id}>
              <CardContent className="flex items-start justify-between gap-4 pt-6">
                <div>
                  <p className="font-medium">{connector.name}</p>
                  <p className="text-xs text-muted-foreground">{connector.url}</p>
                  {connector.lastError && (
                    <p className="text-xs text-destructive">{connector.lastError}</p>
                  )}
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={async () => {
                    await deleteMcpConnector(connector.id);
                    await invalidate();
                  }}
                >
                  Disconnect
                </Button>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
      {connectorsQuery.isLoading && presetsQuery.data && (
        <p className="text-xs text-muted-foreground">Checking connected servers…</p>
      )}
    </div>
  );
}
