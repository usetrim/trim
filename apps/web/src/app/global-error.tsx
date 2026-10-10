"use client";

/**
 * Root layout crash boundary. Keep chrome minimal; site_messages may be unreachable.
 */
export default function GlobalError({
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <html lang="en">
      <body
        style={{
          margin: 0,
          fontFamily: "system-ui, sans-serif",
          display: "flex",
          minHeight: "100vh",
          alignItems: "center",
          justifyContent: "center",
          background: "#09090b",
          color: "#fafafa",
        }}
      >
        <main style={{ textAlign: "center", padding: "2rem", maxWidth: "28rem" }}>
          <h1 style={{ fontSize: "1.5rem", fontWeight: 600 }}>Something went wrong</h1>
          <p style={{ marginTop: "0.75rem", color: "#a1a1aa", lineHeight: 1.5 }}>
            An unexpected error occurred. Try again.
          </p>
          <button
            type="button"
            onClick={() => reset()}
            style={{
              marginTop: "1.5rem",
              height: "2.5rem",
              padding: "0 1rem",
              borderRadius: "0.375rem",
              border: "none",
              background: "#fafafa",
              color: "#09090b",
              fontWeight: 500,
              cursor: "pointer",
            }}
          >
            Try again
          </button>
        </main>
      </body>
    </html>
  );
}
