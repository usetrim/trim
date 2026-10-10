-- Do not invent a provider when signup metadata is missing.
-- auth_provider must come from the OAuth provider claim (handle_new_user).

alter table public.profiles
  alter column auth_provider drop default;

comment on column public.profiles.auth_provider is
  'OAuth provider id from signup (google, github, gitlab, or future allow-list). No SQL default.';
