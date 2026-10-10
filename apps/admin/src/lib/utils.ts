import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function formatMoney(cents: number, currency: string, locale: string) {
  const code = (currency || "").trim().toUpperCase();
  if (!code) {
    const missing = process.env.NEXT_PUBLIC_MONEY_CURRENCY_MISSING?.trim();
    throw new Error(missing || "");
  }
  const loc = (locale || "").trim();
  if (!loc) {
    const missing = process.env.NEXT_PUBLIC_MONEY_LOCALE_MISSING?.trim();
    throw new Error(missing || "");
  }
  return new Intl.NumberFormat(loc, {
    style: "currency",
    currency: code,
    minimumFractionDigits: 2,
  }).format(cents / 100);
}

export function requireEnv(name: string): string {
  const value = process.env[name];
  if (!value || value.trim() === "") {
    const missing = process.env.NEXT_PUBLIC_ENV_MISSING_FMT?.trim();
    if (missing?.includes("%s")) {
      throw new Error(missing.replace("%s", name));
    }
    throw new Error(missing || "");
  }
  return value;
}
