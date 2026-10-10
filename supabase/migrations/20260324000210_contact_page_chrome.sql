-- Contact page chrome + nav (mailto subject/body templates). Email address is COMPANY_SUPPORT_EMAIL via auth-providers.

insert into public.site_messages (code, body) values
  ('LANDING_NAV_CONTACT', 'Contact'),
  ('APP_PATH_CONTACT', '/contact'),
  ('CONTACT_PAGE_TITLE', 'Contact · Trim'),
  ('CONTACT_PAGE_HEADING', 'Contact us'),
  ('CONTACT_PAGE_BODY', 'Questions about Trim, billing, enterprise, or security? Email us - clicking a topic opens your mail app with a ready-made message.'),
  ('CONTACT_META_DESCRIPTION', 'Contact Trim support for product help, billing, enterprise sales, or security reports.'),
  ('CONTACT_EMAIL_LABEL', 'Email'),
  ('CONTACT_OPEN_MAIL_CTA', 'Open mail app'),
  ('CONTACT_TOPICS_HEADING', 'Choose a topic'),
  ('CONTACT_TOPIC_GENERAL_LABEL', 'General support'),
  ('CONTACT_TOPIC_GENERAL_DESC', 'Product questions, account help, and feedback.'),
  ('CONTACT_TOPIC_GENERAL_SUBJECT', 'Trim support request'),
  ('CONTACT_TOPIC_GENERAL_BODY', E'Hi Trim team,\n\nI need help with:\n\n- Account / email:\n- What I tried:\n- What I expected:\n\nThanks.'),
  ('CONTACT_TOPIC_BILLING_LABEL', 'Billing'),
  ('CONTACT_TOPIC_BILLING_DESC', 'Plans, receipts, upgrades, and payment issues.'),
  ('CONTACT_TOPIC_BILLING_SUBJECT', 'Trim billing inquiry'),
  ('CONTACT_TOPIC_BILLING_BODY', E'Hi Trim billing team,\n\nI need help with:\n\n- Account / email:\n- Plan or receipt ID (if any):\n- Issue:\n\nThanks.'),
  ('CONTACT_TOPIC_SALES_LABEL', 'Sales / Enterprise'),
  ('CONTACT_TOPIC_SALES_DESC', 'Team seats, contracts, and custom needs.'),
  ('CONTACT_TOPIC_SALES_SUBJECT', 'Trim Enterprise inquiry'),
  ('CONTACT_TOPIC_SALES_BODY', E'Hi Trim sales team,\n\nI am interested in Enterprise for:\n\n- Company:\n- Approximate seats:\n- Use case:\n\nThanks.'),
  ('CONTACT_TOPIC_SECURITY_LABEL', 'Security'),
  ('CONTACT_TOPIC_SECURITY_DESC', 'Vulnerability reports and security concerns.'),
  ('CONTACT_TOPIC_SECURITY_SUBJECT', 'Trim security report'),
  ('CONTACT_TOPIC_SECURITY_BODY', E'Hi Trim security team,\n\nI would like to report:\n\n- Summary:\n- Impact:\n- Steps to reproduce (no secrets):\n\nThanks.')
on conflict (code) do update set body = excluded.body, updated_at = now();
