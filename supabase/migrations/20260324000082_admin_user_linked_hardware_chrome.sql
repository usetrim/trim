-- Hardware-linked abuse accounts chrome (JA4 linked already exists).
-- User profile nested lists are capped by billing_settings.chart_top_n (workspaces, identities).

insert into public.site_messages (code, body) values
  ('ADMIN_USER_SECTION_LINKED_HW', 'Linked by hardware'),
  ('ADMIN_USER_COL_LINKED_HW', 'Linked account'),
  ('ADMIN_CHECKLIST_MIGRATIONS_LINKED_HW', 'Hardware-linked abuse chrome migration applied')
on conflict (code) do update set body = excluded.body;
