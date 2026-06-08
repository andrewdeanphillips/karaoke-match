import type { MatchResult } from "../types/match";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL;

// MatchError carries the response status alongside the backend's message, so
// callers can distinguish a user-correctable problem (e.g. an invalid
// playlist URL, reported as 400) from a failure that's out of the user's
// hands (a 500, or the request failing outright) — a distinction that's
// lost if only the message text survives.
export class MatchError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

export async function matchPlaylist(url: string): Promise<MatchResult> {
  const response = await fetch(`${API_BASE_URL}/playlist/match`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ url }),
  });

  if (!response.ok) {
    throw new MatchError(response.status, await response.text());
  }

  return response.json();
}
