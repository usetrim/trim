-- Scrub user-facing chrome that named proprietary compression internals
-- (AST / LLMLingua / heuristics / Tree-sitter build tags). Outcome-only copy.

update public.site_messages
set body = 'Start with Fast Mode for instant local compression. Turn on Deep Mode when you want heavier compression on your machine.',
    updated_at = now()
where code = 'LANDING_WHY_4_BODY';

update public.site_messages
set body = 'Local Fast Mode plus optional Deep Mode on your machine. Cloud meters usage and billing only.',
    updated_at = now()
where code = 'LANDING_TAGLINE'
  and body ilike '%AST%';

update public.site_messages
set body = '// src/auth/session.ts (kept)
export async function requireUser(req) {
  const session = await getSession(req);
  if (!session) throw new AuthError("unauthorized");
  return session.user;
}

// vendor/react: reduced to a short outline
// createElement() { /* omitted */ }

// build logs: pruned
// [noise lines removed]',
    updated_at = now()
where code = 'LANDING_DEMO_AFTER_BODY'
  and body ilike '%skeletonized%';

-- Preferences / engine labels (may exist from SeedAndRefresh).
update public.site_messages set body = 'Deep Mode runs only on your machine via trim compress. Compression engines are never loaded in the cloud.', updated_at = now()
where code = 'PREFERENCES_NOTE';
update public.site_messages set body = 'Live proxy (trim start / trim proxy) always runs Fast Mode. Deep Mode runs only via trim compress on your machine. Cloud never loads Deep Mode.', updated_at = now()
where code = 'PREFERENCES_PAGE_DESCRIPTION';
update public.site_messages set body = 'Off = Fast (instant, local). On = Deep (stronger, local only).', updated_at = now()
where code = 'PREFERENCES_DEEP_HINT';
update public.site_messages set body = 'trim compress --bootstrap installs Deep Mode dependencies once on your machine.', updated_at = now()
where code = 'PREFERENCES_CLI_HELP_3';

update public.site_messages set body = 'Deep engine v2 (default)', updated_at = now() where code = 'ENGINE_V2_LABEL';
update public.site_messages set body = 'Deep engine long (question-aware)', updated_at = now() where code = 'ENGINE_LONG_LABEL';
update public.site_messages set body = 'Deep engine v1', updated_at = now() where code = 'ENGINE_V1_LABEL';

update public.site_messages set body = 'This .trimrc custom rule needs the enhanced Trim build. The default install cannot apply those custom queries.', updated_at = now()
where code = 'CLI_TREESITTER_REQUIRED';

update public.site_messages set body = 'Installing Deep Mode dependencies locally. This machine only; never in the hosted cloud.', updated_at = now()
where code in ('CLI_DEEP_BOOTSTRAP_REQS', 'CLI_DEEP_BOOTSTRAP_PIP');
update public.site_messages set body = 'Deep Mode requirements file not found beside the CLI. Pack it with the release or set TRIM_OPTIMIZER_PY to a tree that includes the requirements file.', updated_at = now()
where code = 'CLI_DEEP_REQUIREMENTS_MISSING';
update public.site_messages set body = 'Deep Mode runtime not found. Installing locally once...', updated_at = now()
where code = 'CLI_DEEP_AUTO_INSTALL';
update public.site_messages set body = 'Deep Mode runtime not found beside the CLI (pack it with the release or set TRIM_OPTIMIZER_PY).', updated_at = now()
where code = 'CLI_DEEP_PY_MISSING';
update public.site_messages set body = 'Deep Mode dependencies missing (%v): %s; run: trim compress --bootstrap', updated_at = now()
where code = 'CLI_DEEP_LLM_MISSING_FMT';
update public.site_messages set body = 'Installing Deep Mode dependencies failed', updated_at = now()
where code = 'CLI_DEEP_PIP_FAILED';

update public.site_messages set body = 'Local context optimization proxy for AI coding tools', updated_at = now()
where code = 'CLI_HELP_ROOT_SHORT';
update public.site_messages set body = 'Trim intercepts IDE-to-model traffic and shrinks noisy context on your machine to cut token costs.', updated_at = now()
where code = 'CLI_HELP_ROOT_LONG';
update public.site_messages set body = 'Starts the local Fast Mode proxy. Deep Mode is available via trim compress, not on every proxied IDE request.', updated_at = now()
where code = 'CLI_HELP_START_LONG';
update public.site_messages set body = 'Compress a file with Fast or Deep Mode (Deep runs locally only)', updated_at = now()
where code = 'CLI_HELP_COMPRESS_SHORT';
update public.site_messages set body = 'Fast Mode is low-latency local compression. Deep Mode runs stronger local compression on your machine only (never in the hosted cloud).', updated_at = now()
where code = 'CLI_HELP_COMPRESS_LONG';
update public.site_messages set body = 'question required when Deep engine is long', updated_at = now()
where code = 'CLI_HELP_COMPRESS_FLAG_QUESTION';
update public.site_messages set body = 'install Deep Mode dependencies locally', updated_at = now()
where code = 'CLI_HELP_COMPRESS_FLAG_BOOTSTRAP';

-- Plan marketing features: drop "AST" from public plan cards.
update public.plan_catalog
set features = replace(features::text, 'Local AST proxy', 'Local Fast Mode proxy')::jsonb
where features::text like '%Local AST proxy%';

-- Legal leftover overshare (if any pre-126 rows somehow remain).
update public.site_legal_sections
set body = replace(body, 'Deep Mode (LLMLingua family) runs only on trim compress on the developer machine; the hosted cloud path does not load those models.',
                   'Deep Mode runs only via trim compress on your machine; the hosted cloud path does not load Deep Mode.'),
    updated_at = now()
where body like '%LLMLingua%';

update public.site_legal_sections
set body = replace(body, 'Fast Mode heuristics', 'local compression features'),
    updated_at = now()
where body like '%Fast Mode heuristics%';

update public.site_messages set body = 'Local proxy live meter. Side-by-side shows the last request before and after Trim. Active mode: %s.', updated_at = now()
where code = 'LOCAL_LEAD_FMT';
update public.site_messages set body = 'balanced (default Fast)', updated_at = now()
where code = 'LOCAL_MODE_BALANCED';
update public.site_messages set body = 'aggressive (stronger Fast)', updated_at = now()
where code = 'LOCAL_MODE_AGGRESSIVE';
update public.site_messages set body = 'Paste a prompt or markdown code fence, pick a compression mode, and preview what Trim would forward. Uses the same Fast Mode path as the live proxy.', updated_at = now()
where code = 'LOCAL_PLAYGROUND_LEAD';

update public.site_messages set body = 'Enhanced grammar build required', updated_at = now()
where code = 'ADMIN_PRODUCT_TREESITTER';
