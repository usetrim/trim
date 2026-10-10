-- Professional plan vs top-up dialog chrome (user-facing; no operator/Paddle jargon).
insert into public.site_messages (code, body) values
  ('PLANS_DIALOG_TITLE', 'Choose your Trim plan'),
  ('PLANS_DIALOG_DESCRIPTION', 'Pick a plan that fits how you use Trim. Switch between monthly and annual billing anytime before checkout.'),
  ('PLANS_DIALOG_UPGRADES_ONLY_NOTE', 'You can upgrade to a higher plan. Downgrades are not available while your current plan is active.'),
  ('TOPUP_DIALOG_TITLE', 'Buy credit top-up'),
  ('TOPUP_DIALOG_DESCRIPTION', 'Add cloud credits to your account without changing your plan. Credits apply after checkout completes.'),
  ('TOPUP_DIALOG_EMPTY', 'No top-up packs are available right now. Check back later or upgrade your plan instead.'),
  ('CHANGE_BLOCKED_DEFAULT', 'That plan change is not available for your account right now.'),
  ('PRORATION_REVIEW_HINT', 'Your bill will be adjusted for unused time on your current plan. Review the totals below before confirming.'),
  ('UPGRADE_PRORATION_MODE_FMT', 'Your bill will be adjusted for unused time on your current plan. Review the totals below before confirming.')
on conflict (code) do update set body = excluded.body;
