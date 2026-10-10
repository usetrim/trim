export type AdminNavItem = {
  /** Stable admin_nav_items.id - used for code-only icon mapping (never an icon from DB). */
  id?: string;
  href?: string;
  code?: string;
  perm?: string;
  label?: string;
};

export type AdminNavSection = {
  id?: string;
  code?: string;
  label?: string;
  items?: AdminNavItem[];
};

export type AdminMe = {
  user_id: string;
  role_id: string;
  role_slug: string;
  is_owner: boolean;
  permissions: string[];
  brand: string;
  tagline: string;
  totp_enrolled?: boolean;
  totp_key_ready?: boolean;
  webauthn_enrolled?: boolean;
  webauthn_rp_ready?: boolean;
  nav?: AdminNavItem[];
  nav_sections?: AdminNavSection[];
  /** Shell chrome from /me (sign-out, theme, menu). Fail-closed empty strings. */
  chrome?: Record<string, string>;
};
