import { createFileRoute, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/dashboard/insights/sources")({
  beforeLoad: () => {
    throw redirect({ to: "/dashboard/insights" });
  },
});
