import type { MatchResult } from "../types/match";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL;

export async function matchPlaylist(url: string): Promise<MatchResult> {
  const response = await fetch(`${API_BASE_URL}/playlist/match`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ url }),
  });

  if (!response.ok) {
    throw new Error(await response.text());
  }

  return response.json();
}
