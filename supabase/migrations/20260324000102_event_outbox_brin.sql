-- Durable async trim_events outbox (multi-replica safe via SKIP LOCKED).
-- BRIN on created_at for time-range dashboard scans (monthly RANGE partitions added in 106).

create table if not exists public.trim_event_outbox (
  id bigserial primary key,
  user_id uuid not null,
  request_id text,
  model text,
  tokens_before int not null default 0,
  tokens_after int not null default 0,
  latency_ms int not null default 0,
  mode text not null,
  status text not null,
  error_code text,
  created_at timestamptz not null default now()
);

create index if not exists idx_trim_event_outbox_id on public.trim_event_outbox (id);

comment on table public.trim_event_outbox is
  'Durable queue for proxy/async trim_events inserts; workers claim with FOR UPDATE SKIP LOCKED';

create index if not exists idx_trim_events_created_at_brin
  on public.trim_events using brin (created_at);

comment on index public.idx_trim_events_created_at_brin is
  'Time-range scans for usage/heatmap/retention; complements monthly RANGE partitions';
