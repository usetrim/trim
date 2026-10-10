-- Product default: Deep Mode on (DEFAULT_COMPRESSION_TIER = deep).
-- Signup (handle_new_user) and preferences UI read this from site_messages; no invent.
-- Aligns SeedAndRefresh extras with live product intent. Autostart + DEFAULT_DEEP_ENGINE
-- already default on/selected (true / v2).

do $$
declare
  fast_tier text;
  deep_tier text;
begin
  select lower(btrim(m.body)) into fast_tier
  from public.site_messages m
  where m.code = 'COMPRESSION_TIER_FAST';

  select lower(btrim(m.body)) into deep_tier
  from public.site_messages m
  where m.code = 'COMPRESSION_TIER_DEEP';

  if fast_tier is null or fast_tier = '' or deep_tier is null or deep_tier = '' then
    raise exception 'site_messages COMPRESSION_TIER_FAST / COMPRESSION_TIER_DEEP missing';
  end if;

  update public.site_messages
  set body = deep_tier, updated_at = now()
  where code = 'DEFAULT_COMPRESSION_TIER'
    and lower(btrim(body)) is distinct from deep_tier;

  if not exists (
    select 1 from public.site_messages where code = 'DEFAULT_COMPRESSION_TIER'
  ) then
    insert into public.site_messages (code, body) values ('DEFAULT_COMPRESSION_TIER', deep_tier);
  end if;

  -- Backfill accounts still on the previous signup default (fast) to the new product default.
  update public.profiles
  set compression_tier = deep_tier, updated_at = now()
  where lower(btrim(compression_tier)) = fast_tier;
end;
$$;
