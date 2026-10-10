"use client";

import { TeamClient } from "@/components/team/team-client";
import { useDashboardSession } from "@/hooks/use-dashboard-session";

export default function TeamPage() {
  const { accessToken, userId } = useDashboardSession();
  return <TeamClient accessToken={accessToken} userId={userId} />;
}
