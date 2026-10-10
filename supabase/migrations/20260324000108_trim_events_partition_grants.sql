-- Grants + idempotent harden for trim_events partition helpers (API roles).
grant execute on function public.ensure_trim_events_month_partitions(int, int) to postgres, service_role;
grant execute on function public.drop_trim_events_partitions_older_than(int) to postgres, service_role;
