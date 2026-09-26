import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { BookOpen, Plug, Plus, RefreshCw, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { useToast } from "@/hooks/use-toast";
import {
  createMcpConnector,
  deleteMcpConnector,
  listAgentTools,
  listMcpConnectors,
  refreshMcpConnector,
  updateMcpConnector,
} from "@/services/connectors";
import {
  createAgentSkill,
  deleteAgentSkill,
  listAgentSkills,
  updateAgentSkill,
} from "@/services/skills";

export function McpConnectorsSettings() {
  return (
    <div className="space-y-6">
      <SkillsSettings />
      <ConnectorsSettings />
      <HarnessTools />
    </div>
  );
}

function SkillsSettings() {
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [instructions, setInstructions] = useState("");

  const skillsQuery = useQuery({
    queryKey: ["agent-skills"],
    queryFn: listAgentSkills,
  });

  const invalidate = async () => {
    await queryClient.invalidateQueries({ queryKey: ["agent-skills"] });
  };

  const createMutation = useMutation({
    mutationFn: createAgentSkill,
    onSuccess: async () => {
      setName("");
      setDescription("");
      setInstructions("");
      await invalidate();
      toast({ title: "Skill added" });
    },
    onError: (error: Error) => {
      toast({
        title: "Could not add skill",
        description: error.message,
        variant: "destructive",
      });
    },
  });

  return (
    <div className="space-y-4">
      <div className="space-y-1">
        <p className="font-medium flex items-center gap-2">
          <BookOpen className="h-4 w-4" />
          Custom skills
        </p>
        <p className="text-sm text-muted-foreground">
          Named instruction packs the writing agent follows when they apply. Turn a skill off to keep it without using it.
        </p>
      </div>

      <div className="space-y-2 rounded-lg border p-3">
        <div className="grid gap-2">
          <Label htmlFor="skill-name">Name</Label>
          <Input
            id="skill-name"
            value={name}
            onChange={(event) => setName(event.target.value)}
            placeholder="Editorial voice"
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="skill-when">When to use</Label>
          <Input
            id="skill-when"
            value={description}
            onChange={(event) => setDescription(event.target.value)}
            placeholder="Keep a calm, informational tone"
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="skill-instructions">Instructions</Label>
          <Textarea
            id="skill-instructions"
            value={instructions}
            onChange={(event) => setInstructions(event.target.value)}
            placeholder="Avoid hype. Prefer concrete claims and the author's existing voice."
            className="min-h-[88px]"
          />
        </div>
        <Button
          size="sm"
          disabled={!name.trim() || !instructions.trim() || createMutation.isPending}
          onClick={() => {
            createMutation.mutate({
              name: name.trim(),
              description: description.trim(),
              instructions: instructions.trim(),
              enabled: true,
            });
          }}
        >
          <Plus className="h-4 w-4 mr-1" />
          Add skill
        </Button>
      </div>

      <div className="space-y-2">
        {(skillsQuery.data?.skills ?? []).map((skill) => (
          <div key={skill.id} className="rounded-lg border p-3 space-y-2">
            <div className="flex items-start justify-between gap-2">
              <div>
                <p className="font-medium text-sm">{skill.name}</p>
                {skill.description && (
                  <p className="text-xs text-muted-foreground">{skill.description}</p>
                )}
                <p className="text-xs text-muted-foreground mt-1 whitespace-pre-wrap line-clamp-3">
                  {skill.instructions}
                </p>
              </div>
              <Switch
                checked={skill.enabled}
                onCheckedChange={async (enabled) => {
                  await updateAgentSkill(skill.id, { enabled });
                  await invalidate();
                }}
              />
            </div>
            <Button
              size="sm"
              variant="ghost"
              onClick={async () => {
                await deleteAgentSkill(skill.id);
                await invalidate();
              }}
            >
              <Trash2 className="h-3.5 w-3.5 mr-1" />
              Remove
            </Button>
          </div>
        ))}
      </div>
    </div>
  );
}

function ConnectorsSettings() {
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const [name, setName] = useState("");
  const [transport, setTransport] = useState("http");
  const [url, setUrl] = useState("");
  const [command, setCommand] = useState("");

  const connectorsQuery = useQuery({
    queryKey: ["mcp-connectors"],
    queryFn: listMcpConnectors,
  });

  const invalidate = async () => {
    await queryClient.invalidateQueries({ queryKey: ["mcp-connectors"] });
    await queryClient.invalidateQueries({ queryKey: ["agent-tools"] });
  };

  const createMutation = useMutation({
    mutationFn: createMcpConnector,
    onSuccess: async () => {
      setName("");
      setUrl("");
      setCommand("");
      await invalidate();
      toast({ title: "Connector added" });
    },
    onError: (error: Error) => {
      toast({
        title: "Could not add connector",
        description: error.message,
        variant: "destructive",
      });
    },
  });

  return (
    <div className="space-y-4">
      <div className="space-y-1">
        <p className="font-medium flex items-center gap-2">
          <Plug className="h-4 w-4" />
          MCP connectors
        </p>
        <p className="text-sm text-muted-foreground">
          Add stdio, SSE, or HTTP MCP servers. Their tools join the writing agent harness.
        </p>
      </div>

      <div className="space-y-2 rounded-lg border p-3">
        <div className="grid gap-2">
          <Label htmlFor="mcp-name">Name</Label>
          <Input
            id="mcp-name"
            value={name}
            onChange={(event) => setName(event.target.value)}
            placeholder="Docs search"
          />
        </div>
        <div className="grid gap-2">
          <Label>Transport</Label>
          <Select value={transport} onValueChange={setTransport}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="http">HTTP</SelectItem>
              <SelectItem value="sse">SSE</SelectItem>
              <SelectItem value="stdio">stdio</SelectItem>
            </SelectContent>
          </Select>
        </div>
        {transport === "stdio" ? (
          <div className="grid gap-2">
            <Label htmlFor="mcp-command">Command</Label>
            <Input
              id="mcp-command"
              value={command}
              onChange={(event) => setCommand(event.target.value)}
              placeholder="npx -y @example/mcp-server"
            />
          </div>
        ) : (
          <div className="grid gap-2">
            <Label htmlFor="mcp-url">URL</Label>
            <Input
              id="mcp-url"
              value={url}
              onChange={(event) => setUrl(event.target.value)}
              placeholder="https://example.com/mcp"
            />
          </div>
        )}
        <Button
          size="sm"
          disabled={!name.trim() || createMutation.isPending}
          onClick={() => {
            const parts = command.trim().split(/\s+/).filter(Boolean);
            createMutation.mutate({
              name: name.trim(),
              transport,
              command: parts[0] || command,
              url,
              args: parts.slice(1),
              enabled: true,
            });
          }}
        >
          <Plus className="h-4 w-4 mr-1" />
          Add connector
        </Button>
      </div>

      <div className="space-y-2">
        {(connectorsQuery.data?.connectors ?? []).map((connector) => (
          <div key={connector.id} className="rounded-lg border p-3 space-y-2">
            <div className="flex items-start justify-between gap-2">
              <div>
                <p className="font-medium text-sm">{connector.name}</p>
                <p className="text-xs text-muted-foreground">
                  {connector.transport}
                  {connector.url ? ` · ${connector.url}` : ""}
                  {connector.command ? ` · ${connector.command}` : ""}
                </p>
                {connector.lastError && (
                  <p className="text-xs text-destructive mt-1">{connector.lastError}</p>
                )}
              </div>
              <Switch
                checked={connector.enabled}
                onCheckedChange={async (enabled) => {
                  await updateMcpConnector(connector.id, { enabled });
                  await invalidate();
                }}
              />
            </div>
            <div className="flex gap-2">
              <Button
                size="sm"
                variant="outline"
                onClick={async () => {
                  try {
                    const result = await refreshMcpConnector(connector.id);
                    await invalidate();
                    toast({
                      title: "Tools refreshed",
                      description: result.toolNames?.join(", ") || "No tools advertised",
                    });
                  } catch (error) {
                    toast({
                      title: "Refresh failed",
                      description: error instanceof Error ? error.message : "Could not reach MCP server",
                      variant: "destructive",
                    });
                  }
                }}
              >
                <RefreshCw className="h-3.5 w-3.5 mr-1" />
                Refresh
              </Button>
              <Button
                size="sm"
                variant="ghost"
                onClick={async () => {
                  await deleteMcpConnector(connector.id);
                  await invalidate();
                }}
              >
                <Trash2 className="h-3.5 w-3.5 mr-1" />
                Remove
              </Button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function HarnessTools() {
  const toolsQuery = useQuery({
    queryKey: ["agent-tools"],
    queryFn: listAgentTools,
  });
  const skillsQuery = useQuery({
    queryKey: ["agent-skills"],
    queryFn: listAgentSkills,
  });
  const activeSkills = (skillsQuery.data?.skills ?? []).filter((skill) => skill.enabled);

  return (
    <div className="space-y-2">
      <div className="space-y-1">
        <p className="text-sm font-medium">Active skills</p>
        <div className="flex flex-wrap gap-1">
          {activeSkills.length === 0 ? (
            <span className="text-[11px] text-muted-foreground">None activated</span>
          ) : (
            activeSkills.map((skill) => (
              <span
                key={skill.id}
                className="rounded-full border px-2 py-0.5 text-[11px] text-muted-foreground"
              >
                {skill.name}
              </span>
            ))
          )}
        </div>
      </div>
      <div className="space-y-1">
        <p className="text-sm font-medium">Harness tools</p>
        <div className="flex flex-wrap gap-1">
          {(toolsQuery.data?.tools ?? []).map((tool) => (
            <span
              key={tool.name}
              className="rounded-full border px-2 py-0.5 text-[11px] text-muted-foreground"
            >
              {tool.name}
            </span>
          ))}
        </div>
      </div>
    </div>
  );
}
