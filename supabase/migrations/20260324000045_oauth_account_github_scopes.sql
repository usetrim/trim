-- GitHub profile + private email scopes (Supabase GitHub provider best practice).
-- Settings account chrome (signed-in provider is backend-driven, never invented).
-- Fail-closed login copy when an OAuth identity has no email.

update public.site_messages
set body = 'read:user user:email'
where code = 'AUTH_GITHUB_OAUTH_SCOPES';

insert into public.site_messages (code, body) values
  ('PREFERENCES_ACCOUNT_TITLE', 'Signed-in account'),
  ('PREFERENCES_AUTH_PROVIDER_PREFIX', 'Signed in with'),
  ('LOGIN_OAUTH_EMAIL_MISSING', 'This sign-in provider did not share an email address. Grant email access and try again.'),
  ('ACCOUNT_AUTH_PROVIDER_MISSING', 'Signed-in account provider is missing from your profile.')
on conflict (code) do nothing;

comment on column public.site_messages.body is
  'Operator-owned chrome. AUTH_GITHUB_OAUTH_SCOPES should stay read:user user:email so GitHub name, avatar, and private email reach Supabase.';
