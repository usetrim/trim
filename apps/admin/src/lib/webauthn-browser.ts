/**
 * Browser WebAuthn helpers. Options come from the Go API (go-webauthn JSON).
 * No invented RP id or challenge; fail closed when PublicKeyCredential is missing.
 */

function bufferToBase64URL(buf: ArrayBuffer): string {
  const bytes = new Uint8Array(buf);
  let str = "";
  for (const b of bytes) str += String.fromCharCode(b);
  return btoa(str).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

function base64URLToBuffer(value: string): ArrayBuffer {
  const pad = "=".repeat((4 - (value.length % 4)) % 4);
  const b64 = (value + pad).replace(/-/g, "+").replace(/_/g, "/");
  const str = atob(b64);
  const bytes = new Uint8Array(str.length);
  for (let i = 0; i < str.length; i++) bytes[i] = str.charCodeAt(i);
  return bytes.buffer;
}

function reviveCreateOptions(options: Record<string, unknown>): CredentialCreationOptions {
  const root =
    options.publicKey && typeof options.publicKey === "object" ? options : { publicKey: options };
  const publicKey = {
    ...(root.publicKey as Record<string, unknown>),
  } as unknown as PublicKeyCredentialCreationOptions;
  const challenge = publicKey.challenge as unknown;
  if (typeof challenge !== "string" || !challenge.trim()) {
    throw new Error("unsupported");
  }
  publicKey.challenge = base64URLToBuffer(challenge);
  const user = publicKey.user as PublicKeyCredentialUserEntity | undefined;
  if (!user || typeof (user.id as unknown) !== "string") {
    throw new Error("unsupported");
  }
  publicKey.user = {
    ...user,
    id: base64URLToBuffer(user.id as unknown as string),
  };
  if (Array.isArray(publicKey.excludeCredentials)) {
    publicKey.excludeCredentials = publicKey.excludeCredentials.map((c) => ({
      ...c,
      id: typeof c.id === "string" ? base64URLToBuffer(c.id) : (c.id as BufferSource),
    }));
  }
  return { publicKey };
}

function reviveRequestOptions(options: Record<string, unknown>): CredentialRequestOptions {
  const root =
    options.publicKey && typeof options.publicKey === "object" ? options : { publicKey: options };
  const publicKey = {
    ...(root.publicKey as Record<string, unknown>),
  } as unknown as PublicKeyCredentialRequestOptions;
  const challenge = publicKey.challenge as unknown;
  if (typeof challenge !== "string" || !challenge.trim()) {
    throw new Error("unsupported");
  }
  publicKey.challenge = base64URLToBuffer(challenge);
  if (Array.isArray(publicKey.allowCredentials)) {
    publicKey.allowCredentials = publicKey.allowCredentials.map((c) => ({
      ...c,
      id: typeof c.id === "string" ? base64URLToBuffer(c.id) : (c.id as BufferSource),
    }));
  }
  return { publicKey };
}

function credentialToJSON(cred: PublicKeyCredential): Record<string, unknown> {
  const anyCred = cred as PublicKeyCredential & {
    toJSON?: () => Record<string, unknown>;
  };
  if (typeof anyCred.toJSON === "function") {
    // DOM lib JSON shapes lack an index signature; API body is a plain object.
    return anyCred.toJSON();
  }
  const response = cred.response;
  if (response instanceof AuthenticatorAttestationResponse) {
    return {
      id: cred.id,
      rawId: bufferToBase64URL(cred.rawId),
      type: cred.type,
      response: {
        clientDataJSON: bufferToBase64URL(response.clientDataJSON),
        attestationObject: bufferToBase64URL(response.attestationObject),
        transports: typeof response.getTransports === "function" ? response.getTransports() : [],
      },
      clientExtensionResults: cred.getClientExtensionResults?.() ?? {},
    };
  }
  if (response instanceof AuthenticatorAssertionResponse) {
    return {
      id: cred.id,
      rawId: bufferToBase64URL(cred.rawId),
      type: cred.type,
      response: {
        clientDataJSON: bufferToBase64URL(response.clientDataJSON),
        authenticatorData: bufferToBase64URL(response.authenticatorData),
        signature: bufferToBase64URL(response.signature),
        userHandle: response.userHandle ? bufferToBase64URL(response.userHandle) : null,
      },
      clientExtensionResults: cred.getClientExtensionResults?.() ?? {},
    };
  }
  throw new Error("unsupported");
}

export function webauthnSupported(): boolean {
  return (
    typeof window !== "undefined" &&
    typeof window.PublicKeyCredential !== "undefined" &&
    typeof navigator.credentials?.create === "function" &&
    typeof navigator.credentials?.get === "function"
  );
}

export async function createPasskey(
  options: Record<string, unknown>,
): Promise<Record<string, unknown>> {
  if (!webauthnSupported()) {
    throw new Error("unsupported");
  }
  const cred = (await navigator.credentials.create(
    reviveCreateOptions(options),
  )) as PublicKeyCredential | null;
  if (!cred) {
    throw new Error("cancelled");
  }
  return credentialToJSON(cred);
}

export async function assertPasskey(
  options: Record<string, unknown>,
): Promise<Record<string, unknown>> {
  if (!webauthnSupported()) {
    throw new Error("unsupported");
  }
  const cred = (await navigator.credentials.get(
    reviveRequestOptions(options),
  )) as PublicKeyCredential | null;
  if (!cred) {
    throw new Error("cancelled");
  }
  return credentialToJSON(cred);
}
