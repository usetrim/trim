-- User compression preferences (Fast Go heuristics vs Deep local LLMLingua).
-- Synced to CLI via dashboard settings; cloud never runs Deep models.

alter table public.profiles
  add column if not exists compression_tier text not null default 'fast',
  add column if not exists deep_engine text not null default 'v2',
  add column if not exists deep_target_token int not null default 300;

do $$
begin
  if not exists (
    select 1 from pg_constraint where conname = 'profiles_compression_tier_check'
  ) then
    alter table public.profiles
      add constraint profiles_compression_tier_check
      check (compression_tier in ('fast', 'deep'));
  end if;
  if not exists (
    select 1 from pg_constraint where conname = 'profiles_deep_engine_check'
  ) then
    alter table public.profiles
      add constraint profiles_deep_engine_check
      check (deep_engine in ('v1', 'long', 'v2'));
  end if;
  if not exists (
    select 1 from pg_constraint where conname = 'profiles_deep_target_token_check'
  ) then
    alter table public.profiles
      add constraint profiles_deep_target_token_check
      check (deep_target_token > 0 and deep_target_token <= 100000);
  end if;
end $$;

comment on column public.profiles.compression_tier is
  'fast = Go heuristics; deep = local LLMLingua on developer machine (never cloud GPU).';
comment on column public.profiles.deep_engine is
  'v1=LLMLingua, long=LongLLMLingua, v2=LLMLingua-2';
