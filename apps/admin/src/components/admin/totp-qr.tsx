"use client";

import { useEffect, useState } from "react";

/** Renders otpauth:// URL as a scannable QR (fail-closed: null when empty/encode fails). */
export function TotpQr({
  otpauthUrl,
  alt,
}: {
  otpauthUrl: string;
  /** Accessible label from existing chrome (e.g. secret label); empty = decorative. */
  alt?: string;
}) {
  const [dataUrl, setDataUrl] = useState("");

  useEffect(() => {
    const url = otpauthUrl.trim();
    if (!url) {
      setDataUrl("");
      return;
    }
    let cancelled = false;
    void import("qrcode")
      .then((QR) =>
        QR.toDataURL(url, {
          errorCorrectionLevel: "M",
          margin: 2,
          width: 192,
          color: { dark: "#000000", light: "#ffffff" },
        }),
      )
      .then((png) => {
        if (!cancelled) setDataUrl(png);
      })
      .catch(() => {
        if (!cancelled) setDataUrl("");
      });
    return () => {
      cancelled = true;
    };
  }, [otpauthUrl]);

  if (!dataUrl) return null;
  return (
    <img
      src={dataUrl}
      alt={(alt || "").trim()}
      width={192}
      height={192}
      className="rounded-md border border-border bg-white p-2"
    />
  );
}
