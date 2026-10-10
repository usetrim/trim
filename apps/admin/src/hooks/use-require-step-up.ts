"use client";

import { useCallback, useState } from "react";

/**
 * Compatibility gate for dangerous admin writes.
 *
 * Server policy (enroll_only): once TOTP or passkey is enrolled, CRUD never
 * re-prompts. Call sites keep `if (!requireStepUp()) return` - always allow.
 * Unenrolled operators are blocked by the API (ADMIN_STEP_UP_FACTOR_REQUIRED);
 * MutationCache opens the enroll dialog once.
 */
export function useRequireStepUp(_token: string) {
  void _token;
  const [err, setErr] = useState("");

  const requireStepUp = useCallback((): boolean => {
    setErr("");
    return true;
  }, []);

  return { err, setErr, requireStepUp, stepUpRequired: "" };
}
