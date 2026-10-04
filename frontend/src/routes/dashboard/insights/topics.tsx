import { createFileRoute, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/dashboard/insights/topics")({
  beforeLoad: () => {
    throw redirect({ to: "/dashboard/insights" });
  },
});
