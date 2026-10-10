-- Idempotent Paddle webhook processing and receipt tax identifier support.

create table if not exists public.paddle_webhook_events (
  event_id text primary key,
  event_type text not null,
  processed_at timestamptz not null default now()
);

comment on table public.paddle_webhook_events is
  'Deduplicates Paddle webhook deliveries by event_id (at-least-once delivery).';

create index if not exists idx_paddle_webhook_events_processed_at
  on public.paddle_webhook_events (processed_at desc);
