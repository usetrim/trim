-- Config rates bounds labels + proxy worker start chrome (DB-driven). Idempotent. No em dashes.

insert into public.site_messages (code, body, updated_at) values
  ('CLI_CONFIG_RATES_MIN_LABEL', 'min', now()),
  ('CLI_CONFIG_RATES_MAX_LABEL', 'max', now()),
  ('CLI_CONFIG_RATES_STEP_LABEL', 'step', now()),
  ('CLI_PROXY_WORKER_START_FMT', 'start proxy worker: %v', now())
on conflict (code) do update
set body = excluded.body,
    updated_at = now();
