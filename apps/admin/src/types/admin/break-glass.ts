export type BreakGlassItem = {
  id: string;
  status?: string;
  reason?: string;
  elevates_permission?: string;
  requester_id?: string;
  approver_id?: string;
  starts_at?: string;
  ends_at?: string;
  created_at?: string;
};

export type BreakGlassCreateBody = {
  reason: string;
  elevates_permission: string;
};
