import { createFileRoute, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/dashboard/tasks/")({
  beforeLoad: () => {
    throw redirect({
      to: "/dashboard/insights",
      search: { tab: "tasks" },
    });
  },
});
