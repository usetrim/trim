-- Clarify product is not fully offline: local compress ≠ no internet.
-- Upstream models and Trim cloud (auth/quotas/billing) still need network.

update public.site_messages
set body = 'Compression runs on your machine. Trim cloud handles login, quotas, and billing; upstream models still need network.',
    updated_at = now()
where code = 'LANDING_HERO_NOTE';

update public.site_messages
set body = 'trim start binds an OpenAI-compatible proxy on localhost. Compression runs locally, then the slim prompt goes to your model provider.',
    updated_at = now()
where code = 'LANDING_HOW_1_BODY';

update public.site_messages
set body = 'Install the local proxy in a minute. Sign in for Trim cloud metering, plans, and team receipts; upstream models still require network.',
    updated_at = now()
where code = 'LANDING_WHY_10_BODY';

update public.site_messages
set body = 'Local proxy compresses on your machine. Sign in for Trim cloud auth, quotas, and paid plans; chat still needs an upstream model.',
    updated_at = now()
where code = 'LANDING_INSTALL_SUBTITLE';

update public.site_messages
set body = 'Start with Fast Mode on the live proxy. Use Deep Mode via trim compress for stronger on-machine file/batch compression.',
    updated_at = now()
where code = 'LANDING_WHY_4_BODY';

update public.site_messages
set body = 'Deep Mode runs via trim compress on your machine (file/batch path, not live proxy). Engines are not loaded in Trim cloud; auth and quotas still use Trim cloud when licensed.',
    updated_at = now()
where code = 'PREFERENCES_NOTE';

update public.site_messages
set body = 'Live proxy always runs Fast Mode, then forwards upstream. Deep Mode is trim compress on your machine. Trim cloud never loads Deep Mode; it still handles auth, quotas, and billing.',
    updated_at = now()
where code = 'PREFERENCES_PAGE_DESCRIPTION';

update public.site_messages
set body = 'Off = Fast (live proxy, local compress). On = Deep (stronger file/batch compress on your machine).',
    updated_at = now()
where code = 'PREFERENCES_DEEP_HINT';

update public.site_messages
set body = 'trim compress --bootstrap installs Deep Mode dependencies on your machine (may download packages).',
    updated_at = now()
where code = 'PREFERENCES_CLI_HELP_3';

update public.site_messages
set body = 'Installing Deep Mode dependencies on this machine (not in Trim cloud).',
    updated_at = now()
where code in ('CLI_DEEP_BOOTSTRAP_REQS', 'CLI_DEEP_BOOTSTRAP_PIP');

update public.site_messages
set body = 'Compress a file with Fast or Deep Mode (on-machine file compression; not the live IDE proxy)',
    updated_at = now()
where code = 'CLI_HELP_COMPRESS_SHORT';

update public.site_messages
set body = 'Fast Mode is low-latency local compression for the live proxy or files. Deep Mode is stronger on-machine file/batch compression; engines are not loaded in Trim cloud. Licensed use still requires Trim login and network.',
    updated_at = now()
where code = 'CLI_HELP_COMPRESS_LONG';

update public.site_messages
set body = 'install Deep Mode dependencies on this machine',
    updated_at = now()
where code = 'CLI_HELP_COMPRESS_FLAG_BOOTSTRAP';

-- Soften overstated tagline leftovers from older scrub.
update public.site_messages
set body = 'Local Fast Mode compression plus optional Deep Mode file compression on your machine. Trim cloud meters auth, quotas, and billing; models still need network.',
    updated_at = now()
where code = 'LANDING_TAGLINE'
  and (
    body ilike '%Cloud meters usage and billing only%'
    or body ilike '%on your machine. Cloud meters%'
  );
