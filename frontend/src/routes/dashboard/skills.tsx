import { useEffect, useState } from "react";
import { createFileRoute } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { BookOpen, Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { useToast } from "@/hooks/use-toast";
import { useAdminDashboard } from "@/services/dashboard/dashboard";
import {
  createAgentSkill,
  deleteAgentSkill,
  listAgentSkills,
  updateAgentSkill,
} from "@/services/skills";

export const Route = createFileRoute("/dashboard/skills")({
  component: SkillsPage,
});

function SkillsPage() {
  const { setPageTitle } = useAdminDashboard();
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [instructions, setInstructions] = useState("");

  useEffect(() => {
    setPageTitle("Skills");
  }, [setPageTitle]);

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

  const skills = skillsQuery.data?.skills ?? [];

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-6 p-6">
      <div className="space-y-1">
        <h1 className="flex items-center gap-2 text-2xl font-semibold tracking-tight">
          <BookOpen className="h-6 w-6" />
          Skills
        </h1>
        <p className="text-sm text-muted-foreground">
          Instruction packs the writing agent follows when they apply. Turn a skill off to keep it without using it.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">New skill</CardTitle>
          <CardDescription>Name the situation, then write the instructions the agent should follow.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
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
              className="min-h-[120px]"
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
            <Plus className="mr-1 h-4 w-4" />
            Add skill
          </Button>
        </CardContent>
      </Card>

      <div className="space-y-3">
        {skills.length === 0 && !skillsQuery.isLoading && (
          <p className="text-sm text-muted-foreground">No skills yet.</p>
        )}
        {skills.map((skill) => (
          <Card key={skill.id}>
            <CardContent className="space-y-3 pt-6">
              <div className="flex items-start justify-between gap-3">
                <div className="space-y-1">
                  <p className="font-medium">{skill.name}</p>
                  {skill.description && (
                    <p className="text-sm text-muted-foreground">{skill.description}</p>
                  )}
                  <p className="whitespace-pre-wrap text-sm text-muted-foreground">{skill.instructions}</p>
                </div>
                <Switch
                  checked={skill.enabled}
                  aria-label={skill.enabled ? `Deactivate ${skill.name}` : `Activate ${skill.name}`}
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
                <Trash2 className="mr-1 h-3.5 w-3.5" />
                Remove
              </Button>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
