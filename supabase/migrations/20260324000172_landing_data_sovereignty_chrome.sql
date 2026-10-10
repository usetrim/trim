-- Bold, honest data-sovereignty marketing on the landing (DB chrome only).
-- Claim: compression runs on-machine; Trim cloud is thin (auth/quotas/billing);
-- the slim prompt still goes upstream to the user's model. Never claim zero egress.

insert into public.site_messages (code, body) values
  (
    'LANDING_TAGLINE',
    'Shrink noisy context on your machine so you pay for signal, not vendor dumps. Trim cloud meters the account; it never needs your raw context pack.'
  ),
  (
    'LANDING_HERO_NOTE',
    'Compression runs on your machine. Trim cloud handles login, quotas, and billing, not your full context. Only the slim prompt you choose goes upstream.'
  ),
  (
    'LANDING_WHY_TITLE',
    'Why teams choose Trim'
  ),
  (
    'LANDING_WHY_SUBTITLE',
    'Local compression. Thin cloud control plane. The IDE tools you already use.'
  ),
  (
    'LANDING_WHY_1_TITLE',
    'Your context stays local'
  ),
  (
    'LANDING_WHY_1_BODY',
    'Fast Mode compresses on your machine before anything is forwarded. Trim cloud meters the account; it does not need the raw pack.'
  ),
  (
    'LANDING_WHY_11_TITLE',
    'Built for regulated teams'
  ),
  (
    'LANDING_WHY_11_BODY',
    'Heavy context is trimmed on the laptop. Cloud sees auth, quotas, and billing, not your private repo dump. Upstream still receives only the slim prompt you send.'
  ),
  (
    'LANDING_FLOW_1_TITLE',
    'Install the local proxy'
  ),
  (
    'LANDING_FLOW_1_BODY',
    'Compression happens on your machine. Run trim start, point your IDE at Trim, and keep the heavy context pack local while trimming.'
  ),
  (
    'LANDING_FLOW_3_BODY',
    'Chat and agents POST through the local proxy. Fast Mode drops vendor dumps, lockfiles, and log spam on your machine; it keeps the files and stacks you are actually editing.'
  ),
  (
    'LANDING_PRICING_SUBTITLE',
    'Start free. Cloud meters entitlements and billing; compression stays on your machine.'
  )
on conflict (code) do update set body = excluded.body, updated_at = now();
