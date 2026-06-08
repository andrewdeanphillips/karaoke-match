const API_BASE_URL = import.meta.env.VITE_API_BASE_URL;

// loginURL starts the Spotify Authorization Code flow. Visitors navigate
// here directly (a plain link, not a fetch) — the backend redirects their
// browser on to Spotify's own consent page, and eventually back again.
export const loginURL = `${API_BASE_URL}/auth/login`;

// hasSpotifySession reports whether the visitor's browser is currently
// carrying a usable Spotify session cookie. It calls a lightweight,
// session-gated endpoint that does no Spotify API work of its own — its only
// job is to say yes or no, so the frontend can decide what to show.
export async function hasSpotifySession(): Promise<boolean> {
  const response = await fetch(`${API_BASE_URL}/auth/session`, {
    credentials: "include",
  });
  return response.ok;
}
