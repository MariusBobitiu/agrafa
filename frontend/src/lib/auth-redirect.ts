const AUTH_REDIRECT_KEY = "agrafa-auth-redirect";

type RedirectLocation = {
  pathname: string;
  search?: string;
  hash?: string;
};

type RedirectState = {
  from?: RedirectLocation;
};

function isSafeRedirect(value: string | null | undefined): value is string {
  return Boolean(value && value.startsWith("/") && !value.startsWith("//"));
}

export function buildRedirectPath(location: RedirectLocation): string {
  return `${location.pathname}${location.search ?? ""}${location.hash ?? ""}`;
}

export function getRedirectCandidateFromState(state: unknown) {
  if (!state || typeof state !== "object") return null;
  const { from } = state as RedirectState;
  if (!from?.pathname) return null;
  return buildRedirectPath(from);
}

export function setAuthRedirect(path: string) {
  if (typeof window === "undefined" || !isSafeRedirect(path)) return;
  window.sessionStorage.setItem(AUTH_REDIRECT_KEY, path);
}

export function consumeAuthRedirect() {
  if (typeof window === "undefined") return null;
  const value = window.sessionStorage.getItem(AUTH_REDIRECT_KEY);
  window.sessionStorage.removeItem(AUTH_REDIRECT_KEY);
  return isSafeRedirect(value) ? value : null;
}

export function clearAuthRedirect() {
  if (typeof window === "undefined") return;
  window.sessionStorage.removeItem(AUTH_REDIRECT_KEY);
}

export function resolveAuthRedirect(options: {
  candidate?: string | null;
  fallback: string;
}) {
  const stored = consumeAuthRedirect();
  if (stored) return stored;
  if (isSafeRedirect(options.candidate)) return options.candidate;
  return options.fallback;
}
