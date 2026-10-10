-- Web app Field descriptions / hints and related chrome (no client invent).
insert into public.site_messages (code, body) values
  (
    'EVENTS_SEARCH',
    'Search traces'
  ),
  (
    'EVENTS_SEARCH_DESC',
    'Filters by model, mode, status, or request id as you type.'
  ),
  (
    'RECEIPTS_PREVIEW_FIELD_DESC',
    'Summary from this receipt. Open the full receipt for line items and PDF.'
  ),
  (
    'WORKSPACE_NAME_DESC',
    'Shown to members and on invite emails. You can rename later.'
  ),
  (
    'WORKSPACE_INVITE_EMAIL_DESC',
    'Invitee must sign in with an allowed social provider using this exact email.'
  ),
  (
    'PREFERENCES_ENGINE_HINT',
    'Chooses which Deep compress engine runs on your machine. Only applies when Deep Mode is on.'
  ),
  (
    'PREFERENCES_TARGET_HINT',
    'Target token budget for Deep Mode compress. Only applies when Deep Mode is on.'
  ),
  (
    'ENTERPRISE_COMPANY_DESC',
    'Legal or trade name we should use when following up.'
  ),
  (
    'ENTERPRISE_MESSAGE_DESC',
    'Include SSO, seat count, timeline, or procurement needs so sales can respond.'
  ),
  (
    'API_KEY_DEVICE_AGENT_CLI',
    'cli'
  ),
  (
    'API_KEY_DEVICE_AGENT_IDE',
    'ide'
  ),
  (
    'API_KEY_DEVICE_AGENT_CI',
    'ci'
  ),
  (
    'EVENTS_DATE_RANGE_DESC',
    'Limits traces to the selected UTC calendar days. Clear to show all.'
  ),
  (
    'WORKSPACE_NAME_LABEL',
    'Workspace name'
  ),
  (
    'WORKSPACE_INVITE_EMAIL_LABEL',
    'Invite email'
  ),
  (
    'RECEIPTS_SEARCH',
    'Search receipts'
  ),
  (
    'RECEIPTS_SEARCH_DESC',
    'Filters by invoice id, status, or bill-to as you type.'
  )
on conflict (code) do update set body = excluded.body;
