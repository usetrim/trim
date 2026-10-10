export type SiteMessageRow = {
  code: string;
  body: string;
};

export type LegalSectionRow = {
  id: string;
  page: string;
  sort_order: number;
  heading: string;
  body: string;
  updated_at?: string;
};
