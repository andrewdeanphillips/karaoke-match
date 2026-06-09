import type { MatchResult } from "../types/match";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL;

export type Example = {
  id: string;
  name: string;
  url: string;
};

type ExamplesResponse = {
  examples: Example[];
};

// fetchExamples lists the curated playlists behind the "try an example" path.
// The backend reports an empty list when that path isn't configured, so
// callers can treat "no examples" and "feature disabled" the same way —
// simply render nothing.
export async function fetchExamples(): Promise<Example[]> {
  const response = await fetch(`${API_BASE_URL}/examples`);
  if (!response.ok) {
    return [];
  }

  const body: ExamplesResponse = await response.json();
  return body.examples;
}

export class ExampleError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

// matchExample runs the playlist-match flow against one of the curated
// example playlists, riding on the one deliberate, owner-held session that
// backs the whole "try an example" path — no Spotify account or login
// required from the visitor.
export async function matchExample(id: string): Promise<MatchResult> {
  const response = await fetch(`${API_BASE_URL}/examples/match`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ id }),
  });

  if (!response.ok) {
    throw new ExampleError(response.status, await response.text());
  }

  return response.json();
}
