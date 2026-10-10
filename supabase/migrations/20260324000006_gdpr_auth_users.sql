-- Strengthen GDPR deletion: anonymize tax ledgers, then remove auth identity.

alter table public.billing_receipts
  drop constraint if exists billing_receipts_user_id_fkey;

alter table public.billing_receipts
  alter column user_id drop not null;

alter table public.billing_receipts
  add constraint billing_receipts_user_id_fkey
  foreign key (user_id) references public.profiles(id) on delete set null;

create or replace function public.execute_gdpr_deletion(target_user_id uuid)
returns void
language plpgsql
security definer
set search_path = public, auth
as $$
begin
  delete from public.api_keys where user_id = target_user_id;
  delete from public.device_fingerprints where user_id = target_user_id;
  delete from public.user_quotas where user_id = target_user_id;
  delete from public.workspace_members where user_id = target_user_id;
  delete from public.subscriptions where user_id = target_user_id;

  update public.billing_receipts
    set user_id = null,
        bill_to_email = 'anonymized@gdpr.deleted',
        bill_to_name = null,
        bill_to_company = null,
        bill_to_address_line1 = null,
        bill_to_address_line2 = null,
        bill_to_city = null,
        bill_to_region = null,
        bill_to_postal_code = null,
        tax_id = null
    where user_id = target_user_id;

  update public.trim_events set user_id = null where user_id = target_user_id;
  delete from public.profiles where id = target_user_id;
  delete from auth.users where id = target_user_id;
end;
$$;
