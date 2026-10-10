export type DashboardKpiItem = {
  id: string;
  label: string;
  value: number;
};

export type DashboardChecklistItem = {
  id: string;
  ok: boolean;
  label: string;
  status_label: string;
};

export type DashboardHealthItem = {
  id: string;
  ok: boolean;
  label: string;
  status_label?: string;
  age_seconds?: number;
  last_at?: string;
  error_count?: number;
  success_count?: number;
};

export type DashboardAlertItem = {
  id: string;
  label: string;
  count: number;
  active: boolean;
};

export type AdminDashboard = {
  kpis: Record<string, number>;
  kpi_items: DashboardKpiItem[];
  health: Record<string, boolean>;
  health_items: DashboardHealthItem[];
  ops_checklist: DashboardChecklistItem[];
  alerts?: DashboardAlertItem[];
  generated_at?: string;
  chrome?: {
    brand?: string;
    tagline?: string;
    ops_checklist_title?: string;
    health_title?: string;
    alerts_title?: string;
    revenue_hint?: string;
    revenue_link?: string;
    revenue_href?: string;
  };
};
